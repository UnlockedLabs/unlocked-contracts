package identity

import (
	"net/http"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

// RequireServiceToken serves next only to requests that carry token as a
// bearer token, and answers 401 to the rest. An empty token rejects every
// request. A product service wraps its routes in it, the same check the
// contracts' own handlers use.
func RequireServiceToken(token string, next http.Handler) http.Handler {
	return rpc.RequireToken(token, next)
}
