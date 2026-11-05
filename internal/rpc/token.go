// Package rpc is what the contracts' HTTP clients and handlers share: the
// service token every internal call carries, and the request and answer
// shape. Callers outside this module use the packages that wrap it.
package rpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/http"
)

// SetToken puts the shared service token on req as a bearer token, the scheme
// every internal call uses.
func SetToken(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

// RequireToken serves next only to requests that carry token as a bearer token, and answers 401
// to the rest. An empty token rejects every request, so a service started
// without one fails closed.
func RequireToken(token string, next http.Handler) http.Handler {
	want := sha256.Sum256([]byte("Bearer " + token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if token == "" || subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			slog.Warn("rpc: refuse a call without the service token", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			WriteJSON(w, http.StatusUnauthorized, Envelope[any]{Message: "missing or wrong token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
