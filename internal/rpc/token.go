// Package rpc is what the contracts' HTTP clients and handlers share: the
// service token every internal call carries, and the request and answer
// shape. Callers outside this module use the packages that wrap it.
package rpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

// TokenHeader carries the shared service token on every internal call.
const TokenHeader = "X-Service-Token"

// RequireToken serves next only to requests that carry token, and answers 401
// to the rest. An empty token rejects every request, so a service started
// without one fails closed.
func RequireToken(token string, next http.Handler) http.Handler {
	want := sha256.Sum256([]byte(token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := sha256.Sum256([]byte(r.Header.Get(TokenHeader)))
		if token == "" || subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			WriteJSON(w, http.StatusUnauthorized, Envelope[any]{Message: "invalid service token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
