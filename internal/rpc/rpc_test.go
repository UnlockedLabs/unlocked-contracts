package rpc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
)

func TestRequireToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	tests := []struct {
		name   string
		want   string
		header *string
		status int
	}{
		{"right token", "s3cret", ptr("Bearer s3cret"), http.StatusNoContent},
		{"token without Bearer", "s3cret", ptr("s3cret"), http.StatusUnauthorized},
		{"missing header", "s3cret", nil, http.StatusUnauthorized},
		{"wrong token", "s3cret", ptr("Bearer other"), http.StatusUnauthorized},
		{"longer token", "s3cret", ptr("Bearer s3cret-and-more"), http.StatusUnauthorized},
		{"empty token configured, empty sent", "", ptr("Bearer "), http.StatusUnauthorized},
		{"empty token configured, none sent", "", nil, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/x", nil)
			if tt.header != nil {
				req.Header.Set("Authorization", *tt.header)
			}
			rec := httptest.NewRecorder()
			rpc.RequireToken(tt.want, next).ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
		})
	}
}

func ptr(s string) *string { return &s }
