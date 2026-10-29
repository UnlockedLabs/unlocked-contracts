package identity_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/features"
	"github.com/UnlockedLabs/unlocked-contracts/identity"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

const (
	kratosID  = "8f2c9a4e-6b1d-4f3a-9c7e-2d5b8a1f0e64"
	sessionID = "c1e5f7a2-3b4d-4e6f-8a9b-0c1d2e3f4a5b"
	csrfToken = "token-123"
)

// The answers a fake Kratos can give to whoami.
const (
	activeSession   = `{"id":"` + sessionID + `","active":true,"identity":{"id":"` + kratosID + `","traits":{"password_reset":false}}}`
	inactiveSession = `{"id":"` + sessionID + `","active":false,"identity":{"id":"` + kratosID + `","traits":{"password_reset":false}}}`
	resetSession    = `{"id":"` + sessionID + `","active":true,"identity":{"id":"` + kratosID + `","traits":{"password_reset":true}}}`
	noResetTrait    = `{"id":"` + sessionID + `","active":true,"identity":{"id":"` + kratosID + `","traits":{}}}`
	stringReset     = `{"id":"` + sessionID + `","active":true,"identity":{"id":"` + kratosID + `","traits":{"password_reset":"false"}}}`
	noIdentityID    = `{"id":"` + sessionID + `","active":true,"identity":{"traits":{"password_reset":false}}}`
)

// coreClaims is what core's /internal/v1/claims answers for kratosID.
const coreClaims = `{"message":"Data fetched successfully","data":{"user_id":12,"username":"jdoe",` +
	`"role":"facility_admin","facility_id":3,"facility_name":"North","feature_access":["open_content","helpful_links"],` +
	`"session_id":"","timezone":"America/Chicago"}}`

// wantClaims are coreClaims with the session's id.
var wantClaims = &identity.Claims{
	UserID: 12, Username: "jdoe", Role: users.FacilityAdmin, FacilityID: 3, FacilityName: "North",
	FeatureAccess: []features.Access{features.OpenContent, features.HelpfulLinks},
	SessionID:     sessionID, TimeZone: "America/Chicago",
}

// fake answers one endpoint with a fixed status and body, and counts its
// calls. check, when set, looks at each request it gets.
type fake struct {
	status int
	body   string
	calls  atomic.Int32
	check  func(*http.Request)
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	if f.check != nil {
		f.check(r)
	}
	w.WriteHeader(f.status)
	_, _ = w.Write([]byte(f.body))
}

func TestRequire(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		cookies      string
		csrfHeader   string
		kratosStatus int
		kratosBody   string
		coreStatus   int
		coreBody     string
		wantStatus   int
		wantMessage  string
		wantKratos   int32 // calls to Kratos
		wantCore     int32 // calls to core
	}{
		{
			name:       "a logged-in user gets core's claims and the session id",
			kratosBody: activeSession, coreBody: coreClaims,
			wantStatus: http.StatusOK, wantKratos: 1, wantCore: 1,
		},
		{
			name:       "no session cookie is turned away without asking Kratos",
			cookies:    "csrf_token=" + csrfToken,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in",
		},
		{
			name:         "a session Kratos rejects is turned away",
			kratosStatus: http.StatusUnauthorized, kratosBody: `{"error":{"code":401}}`,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1,
		},
		{
			name:         "a session Kratos forbids is turned away",
			kratosStatus: http.StatusForbidden, kratosBody: `{"error":{"code":403}}`,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1,
		},
		{
			name:       "an inactive session is turned away",
			kratosBody: inactiveSession,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1,
		},
		{
			name:       "a user core doesn't know is turned away",
			kratosBody: activeSession, coreStatus: http.StatusNotFound, coreBody: `{"message":"user not found","data":null}`,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1, wantCore: 1,
		},
		{
			name:       "a deactivated user is turned away",
			kratosBody: activeSession, coreStatus: http.StatusForbidden, coreBody: `{"message":"account deactivated","data":null}`,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1, wantCore: 1,
		},
		{
			name:       "a user who must reset their password is refused",
			kratosBody: resetSession, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "password reset required", wantKratos: 1, wantCore: 1,
		},
		{
			name:       "a missing password_reset trait counts as a reset to do",
			kratosBody: noResetTrait, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "password reset required", wantKratos: 1, wantCore: 1,
		},
		{
			name:       "a password_reset trait that isn't a bool counts as a reset to do",
			kratosBody: stringReset, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "password reset required", wantKratos: 1, wantCore: 1,
		},
		{
			name:         "Kratos failing is unavailable, not a logout",
			kratosStatus: http.StatusInternalServerError, kratosBody: `oops`,
			wantStatus: http.StatusServiceUnavailable, wantMessage: "login check unavailable, try again", wantKratos: 1,
		},
		{
			name:       "a whoami answer that isn't JSON is unavailable",
			kratosBody: `<html>not json</html>`,
			wantStatus: http.StatusServiceUnavailable, wantMessage: "login check unavailable, try again", wantKratos: 1,
		},
		{
			name:       "a whoami answer with no identity id is unavailable",
			kratosBody: noIdentityID,
			wantStatus: http.StatusServiceUnavailable, wantMessage: "login check unavailable, try again", wantKratos: 1,
		},
		{
			name:       "core failing is unavailable",
			kratosBody: activeSession, coreStatus: http.StatusInternalServerError, coreBody: `{"message":"boom","data":null}`,
			wantStatus: http.StatusServiceUnavailable, wantMessage: "login check unavailable, try again", wantKratos: 1, wantCore: 1,
		},
		{
			name:       "claims with no user id are unavailable",
			kratosBody: activeSession, coreBody: `{"message":"Data fetched successfully","data":{}}`,
			wantStatus: http.StatusServiceUnavailable, wantMessage: "login check unavailable, try again", wantKratos: 1, wantCore: 1,
		},
		{
			name:   "a write with a matching CSRF token goes through",
			method: http.MethodPost, csrfHeader: csrfToken,
			kratosBody: activeSession, coreBody: coreClaims,
			wantStatus: http.StatusOK, wantKratos: 1, wantCore: 1,
		},
		{
			name:       "a write without a CSRF header is refused",
			method:     http.MethodDelete,
			kratosBody: activeSession, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "CSRF token missing", wantKratos: 1, wantCore: 1,
		},
		{
			name:   "a write without a CSRF cookie is refused",
			method: http.MethodPatch, csrfHeader: csrfToken, cookies: "ory_kratos_session=abc",
			kratosBody: activeSession, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "CSRF token missing", wantKratos: 1, wantCore: 1,
		},
		{
			name:   "a write with the wrong CSRF token is refused",
			method: http.MethodPut, csrfHeader: "not-the-token",
			kratosBody: activeSession, coreBody: coreClaims,
			wantStatus: http.StatusForbidden, wantMessage: "CSRF token invalid", wantKratos: 1, wantCore: 1,
		},
		{
			// The login check comes first, as in core.
			name:         "a write that isn't logged in is a 401, not a CSRF error",
			method:       http.MethodPost,
			kratosStatus: http.StatusUnauthorized, kratosBody: `{}`,
			wantStatus: http.StatusUnauthorized, wantMessage: "not logged in", wantKratos: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookies := tt.cookies
			if cookies == "" {
				cookies = "ory_kratos_session=abc; csrf_token=" + csrfToken
			}
			kratos := &fake{status: orOK(tt.kratosStatus), body: tt.kratosBody, check: func(r *http.Request) {
				if r.URL.Path != "/sessions/whoami" {
					t.Errorf("Kratos got path %q, want /sessions/whoami", r.URL.Path)
				}
				if got := r.Header.Get("Cookie"); got != cookies {
					t.Errorf("Kratos got cookies %q, want %q", got, cookies)
				}
			}}
			core := &fake{status: orOK(tt.coreStatus), body: tt.coreBody, check: func(r *http.Request) {
				if r.URL.Path != "/internal/v1/claims" || r.URL.Query().Get("kratos_id") != kratosID {
					t.Errorf("core got %s, want /internal/v1/claims?kratos_id=%s", r.URL, kratosID)
				}
			}}
			kratosSrv, coreSrv := httptest.NewServer(kratos), httptest.NewServer(core)
			defer kratosSrv.Close()
			defer coreSrv.Close()

			var got *identity.Claims
			var reached bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
				got, _ = identity.FromContext(r.Context())
			})
			h := identity.NewSessionChecker(kratosSrv.URL+"/", coreSrv.URL+"/", nil).Require(next)

			req := httptest.NewRequest(orGet(tt.method), "/api/things", nil)
			req.Header.Set("Cookie", cookies)
			if tt.csrfHeader != "" {
				req.Header.Set("X-CSRF-Token", tt.csrfHeader)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body)
			}
			if n := kratos.calls.Load(); n != tt.wantKratos {
				t.Errorf("Kratos was called %d times, want %d", n, tt.wantKratos)
			}
			if n := core.calls.Load(); n != tt.wantCore {
				t.Errorf("core was called %d times, want %d", n, tt.wantCore)
			}
			if tt.wantStatus == http.StatusOK {
				if !reached {
					t.Fatal("next wasn't called")
				}
				if !reflect.DeepEqual(got, wantClaims) {
					t.Errorf("claims = %+v\nwant     %+v", got, wantClaims)
				}
				return
			}
			if reached {
				t.Error("next was called for a turned-away request")
			}
			var body struct {
				Message string `json:"message"`
				Data    any    `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("body isn't core's JSON error shape: %v (%s)", err, w.Body)
			}
			if body.Message != tt.wantMessage || body.Data != nil {
				t.Errorf("body = %+v, want message %q and null data", body, tt.wantMessage)
			}
		})
	}
}

// Kratos or core not running at all: the request is turned away as
// unavailable, never let through.
func TestRequireUnreachable(t *testing.T) {
	up := httptest.NewServer(&fake{status: http.StatusOK, body: activeSession})
	defer up.Close()
	down := httptest.NewServer(http.NotFoundHandler())
	downURL := down.URL
	down.Close()

	tests := []struct {
		name, kratosURL, coreURL string
	}{
		{"Kratos down", downURL, up.URL},
		{"core down", up.URL, downURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("next was called")
			})
			req := httptest.NewRequest(http.MethodGet, "/api/things", nil)
			req.Header.Set("Cookie", "ory_kratos_session=abc")
			w := httptest.NewRecorder()
			identity.NewSessionChecker(tt.kratosURL, tt.coreURL, nil).Require(next).ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
			}
		})
	}
}

func orOK(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

func orGet(method string) string {
	if method == "" {
		return http.MethodGet
	}
	return method
}
