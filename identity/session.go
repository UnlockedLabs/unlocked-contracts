package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// claimsPath is core's endpoint that builds a user's Claims (EN-256). Only
// reachable from inside the cluster.
const claimsPath = "/internal/v1/claims"

// callTimeout bounds each call to Kratos and to core, whatever client the
// SessionChecker was given.
const callTimeout = 2 * time.Second

// The cookies a request carries: Kratos's login session, and the CSRF token
// core sets for the browser to copy into the X-CSRF-Token header.
const (
	sessionCookie = "ory_kratos_session"
	csrfCookie    = "csrf_token"
	csrfHeader    = "X-CSRF-Token"
)

// Why a request was turned away. Any other error means Kratos or core couldn't
// answer.
var (
	errNotLoggedIn   = errors.New("not logged in")
	errPasswordReset = errors.New("password reset required")
)

// notLoggedIn is errNotLoggedIn with the reason, for the log.
func notLoggedIn(reason string) error {
	return fmt.Errorf("%w: %s", errNotLoggedIn, reason)
}

// SessionChecker checks a request's login session the same way core does, for
// a product service whose requests come straight from the ingress instead of
// through core. It is safe for concurrent use.
type SessionChecker struct {
	kratosURL string
	coreURL   string
	client    *http.Client
}

// NewSessionChecker returns a SessionChecker that asks Kratos's public API at
// kratosURL, such as "http://kratos:4433", whether the session is active, and
// core at coreURL, such as "http://server:8080", who the user is. A nil client
// means http.DefaultClient; every call times out after 2 seconds either way.
func NewSessionChecker(kratosURL, coreURL string, client *http.Client) *SessionChecker {
	if client == nil {
		client = http.DefaultClient
	}
	return &SessionChecker{
		kratosURL: strings.TrimSuffix(kratosURL, "/"),
		coreURL:   strings.TrimSuffix(coreURL, "/"),
		client:    client,
	}
}

// Require runs next with the logged-in user's Claims on the request's context,
// the same Claims core builds for them. Like core, it gives a logged-in
// browser a csrf_token cookie when it has none. Otherwise it answers in core's
// JSON error shape:
//   - 401 when no one is logged in: no session cookie, a session Kratos says
//     isn't active, or a user core doesn't know or has deactivated. Logged
//     with slog at Info, with the reason;
//   - 403 when the user must reset their password first, or a request other
//     than GET, HEAD or OPTIONS lacks a matching CSRF token;
//   - 503 when Kratos or core couldn't answer, logged with slog at Error
//     unless the browser had already hung up. It never lets a request
//     through without a check.
func (c *SessionChecker) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := c.claims(r)
		switch {
		case errors.Is(err, errNotLoggedIn):
			slog.InfoContext(r.Context(), "identity: not logged in", "reason", err)
			writeError(w, http.StatusUnauthorized, "not logged in")
			return
		case errors.Is(err, errPasswordReset):
			writeError(w, http.StatusForbidden, "password reset required")
			return
		case err != nil:
			if r.Context().Err() == nil {
				slog.ErrorContext(r.Context(), "identity: check login session", "err", err)
			}
			writeError(w, http.StatusServiceUnavailable, "login check unavailable, try again")
			return
		}
		ensureCSRFCookie(w, r)
		if msg := csrfProblem(r); msg != "" {
			writeError(w, http.StatusForbidden, msg)
			return
		}
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), claims)))
	})
}

// claims asks Kratos about the request's session, then core about its user.
// Core checks the user before the password reset, so a deactivated user is
// told they aren't logged in either way.
func (c *SessionChecker) claims(r *http.Request) (*Claims, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, notLoggedIn("no session cookie")
	}
	session, err := c.whoami(r.Context(), cookie)
	if err != nil {
		return nil, err
	}
	claims, err := c.userClaims(r.Context(), session.Identity.ID)
	if err != nil {
		return nil, err
	}
	// Core treats a missing or non-bool trait as a reset still to do.
	if reset, ok := session.Identity.Traits.PasswordReset.(bool); !ok || reset {
		return nil, errPasswordReset
	}
	claims.SessionID = session.ID
	return claims, nil
}

// session is the part of Kratos's whoami answer the check uses.
type session struct {
	ID       string `json:"id"`
	Active   bool   `json:"active"`
	Identity struct {
		ID     string `json:"id"`
		Traits struct {
			PasswordReset any `json:"password_reset"`
		} `json:"traits"`
	} `json:"identity"`
}

// whoami sends the session cookie to Kratos, which answers 401 for a missing
// or expired session. It sends no other cookie, so it doesn't matter how the
// browser split them across Cookie headers.
func (c *SessionChecker) whoami(ctx context.Context, cookie *http.Cookie) (*session, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.kratosURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("build whoami request: %w", err)
	}
	req.AddCookie(cookie)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ask kratos whoami: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, notLoggedIn("kratos answered " + resp.Status)
	default:
		return nil, fmt.Errorf("kratos whoami answered %s", resp.Status)
	}

	var s session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("read kratos whoami answer: %w", err)
	}
	if !s.Active {
		return nil, notLoggedIn("session isn't active")
	}
	if s.ID == "" || s.Identity.ID == "" {
		return nil, errors.New("kratos whoami answer has no session or identity id")
	}
	return &s, nil
}

// userClaims asks core for the Claims of the user with this Kratos identity.
func (c *SessionChecker) userClaims(ctx context.Context, kratosID string) (*Claims, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	u := c.coreURL + claimsPath + "?kratos_id=" + url.QueryEscape(kratosID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build claims request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ask core for claims: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusForbidden:
		// No such user, or a deactivated one, in core's JSON error. Any other
		// 404, such as a core without this route, is a fault, not a logout.
		var e struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil || e.Message == "" {
			return nil, fmt.Errorf("core answered %s for claims without its JSON error", resp.Status)
		}
		return nil, notLoggedIn("core: " + e.Message)
	default:
		return nil, fmt.Errorf("core answered %s for claims", resp.Status)
	}

	// Core wraps every response in {"message", "data"}.
	var body struct {
		Data Claims `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("read core's claims: %w", err)
	}
	if body.Data.UserID == 0 {
		return nil, errors.New("core's claims have no user id")
	}
	return &body.Data, nil
}

// ensureCSRFCookie gives a browser with no csrf_token cookie a new token, as
// core does for every logged-in request (ensureCSRFToken, with the same
// attributes), so a product's writes don't depend on the browser having
// visited core recently.
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(csrfCookie); err == nil {
		return
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b) // never fails since Go 1.24
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    base64.StdEncoding.EncodeToString(b),
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   12 * 60 * 60,
	})
}

// csrfProblem is core's double-submit check: a request that can change
// something must send the csrf_token cookie's value in the X-CSRF-Token
// header. It returns core's message for a request that fails it, or "".
func csrfProblem(r *http.Request) string {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return ""
	}
	header := r.Header.Get(csrfHeader)
	cookie, err := r.Cookie(csrfCookie)
	if err != nil || header == "" {
		return "CSRF token missing"
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
		return "CSRF token invalid"
	}
	return ""
}

// writeError answers like core's errorResponse: {"message": msg, "data": null}.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{Message: msg})
}
