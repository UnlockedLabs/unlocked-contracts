package identity_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/identity"
)

func TestRequireServiceToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	tests := []struct {
		name, configured, sent string
		want                   int
	}{
		{"right token", "s3cret", "Bearer s3cret", http.StatusNoContent},
		{"wrong token", "s3cret", "Bearer nope", http.StatusUnauthorized},
		{"no scheme", "s3cret", "s3cret", http.StatusUnauthorized},
		{"missing", "s3cret", "", http.StatusUnauthorized},
		{"empty configured token rejects all", "", "Bearer ", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.sent != "" {
				r.Header.Set("Authorization", tt.sent)
			}
			w := httptest.NewRecorder()
			identity.RequireServiceToken(tt.configured, next).ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
