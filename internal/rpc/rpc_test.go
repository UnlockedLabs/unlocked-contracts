package rpc_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

// The 401 body and the logged refusal are what Open Content's handler has
// answered since EN-205, so they must not change.
func TestRequireTokenRefusal(t *testing.T) {
	var logs bytes.Buffer
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))

	rec := httptest.NewRecorder()
	rpc.RequireToken("s3cret", http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", nil))

	if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"missing or wrong token","data":null}` {
		t.Errorf("body = %s, want the missing or wrong token message", got)
	}
	if line := logs.String(); !strings.Contains(line, "path=/x") {
		t.Errorf("log = %q, want the refusal logged with its path", line)
	}
}

func ptr(s string) *string { return &s }
