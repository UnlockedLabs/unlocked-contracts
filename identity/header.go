package identity

import (
	"encoding/json"
	"net/http"
)

// ClaimsHeader carries the caller's Claims as JSON from core's proxy to a
// product service, next to the service token.
const ClaimsHeader = "X-Identity-Claims"

// SetClaimsHeader writes c onto the outgoing request r.
func SetClaimsHeader(r *http.Request, c *Claims) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	r.Header.Set(ClaimsHeader, string(b))
	return nil
}

// ClaimsFromRequest reads the Claims from the incoming request r, and returns
// false when the header is missing or isn't JSON.
func ClaimsFromRequest(r *http.Request) (*Claims, bool) {
	var c Claims
	if err := json.Unmarshal([]byte(r.Header.Get(ClaimsHeader)), &c); err != nil {
		return nil, false
	}
	return &c, true
}
