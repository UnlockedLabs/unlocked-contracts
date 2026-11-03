package features

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	myTool := Feature{Name: "my_tool", Title: "My Tool", Description: "Helps with things"}

	tests := []struct {
		name    string
		status  int
		reply   string
		wantErr string
	}{
		{name: "new feature", status: http.StatusCreated, reply: `{"message":"feature registered","data":{}}`},
		{name: "already registered", status: http.StatusOK, reply: `{"message":"feature registered","data":{}}`},
		{name: "any 2xx", status: http.StatusNoContent},
		{
			name:    "bad name carries core's message",
			status:  http.StatusBadRequest,
			reply:   `{"message":"feature name must start with a letter","data":null}`,
			wantErr: "core answered 400 Bad Request: feature name must start with a letter",
		},
		{
			name:    "core feature name",
			status:  http.StatusConflict,
			reply:   `{"message":"a core feature can't be registered","data":null}`,
			wantErr: "core answered 409 Conflict: a core feature can't be registered",
		},
		{
			name:    "core error",
			status:  http.StatusInternalServerError,
			reply:   `{"message":"error creating feature_flags","data":null}`,
			wantErr: "core answered 500 Internal Server Error: error creating feature_flags",
		},
		{
			name:    "core without the route",
			status:  http.StatusNotFound,
			reply:   "404 page not found",
			wantErr: "core answered 404 Not Found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath, gotType string
			var gotBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath, gotType = r.Method, r.URL.Path, r.Header.Get("Content-Type")
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &gotBody)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.reply))
			}))
			defer srv.Close()

			err := Register(context.Background(), srv.URL+"/", nil, myTool)

			if gotMethod != http.MethodPut || gotPath != "/internal/v1/features/my_tool" || gotType != "application/json" {
				t.Errorf("request = %s %s (%s), want PUT /internal/v1/features/my_tool (application/json)", gotMethod, gotPath, gotType)
			}
			wantBody := map[string]any{"title": "My Tool", "description": "Helps with things"}
			if len(gotBody) != len(wantBody) || gotBody["title"] != wantBody["title"] || gotBody["description"] != wantBody["description"] {
				t.Errorf("body = %v, want %v", gotBody, wantBody)
			}
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Register: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterCoreUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	err := Register(context.Background(), srv.URL, nil, Feature{Name: "my_tool", Title: "My Tool"})
	if err == nil || !strings.HasPrefix(err.Error(), "features: register my_tool:") {
		t.Fatalf("err = %v, want a wrapped error naming the feature", err)
	}
}

func TestRegisterEmptyName(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	err := Register(context.Background(), srv.URL, nil, Feature{Title: "My Tool"})
	if err == nil || err.Error() != "features: register: name is required" {
		t.Fatalf("err = %v, want name is required", err)
	}
	if called {
		t.Fatal("core was called with an empty name")
	}
}

func TestRegisterBadURL(t *testing.T) {
	err := Register(context.Background(), "http://bad host", nil, Feature{Name: "my_tool", Title: "My Tool"})
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Fatalf("err = %v, want a build request error", err)
	}
}

func TestRegisterTimesOut(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	start := time.Now()
	err := Register(context.Background(), srv.URL, nil, Feature{Name: "my_tool", Title: "My Tool"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed < callTimeout || elapsed > callTimeout+time.Second {
		t.Fatalf("took %s, want about %s", elapsed, callTimeout)
	}
}

func TestRegisterCallerCanceled(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Register(ctx, srv.URL, nil, Feature{Name: "my_tool", Title: "My Tool"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
