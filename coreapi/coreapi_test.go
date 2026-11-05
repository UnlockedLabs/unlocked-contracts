package coreapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

const token = "s3cret"

var (
	user    = coreapi.User{ID: 7, Username: "jdoe", NameFirst: "Jane", NameLast: "Doe", Email: "j@x.org", FacilityID: 3}
	newUser = coreapi.NewUser{Username: "jdoe", NameFirst: "Jane", NameLast: "Doe", Email: "j@x.org", FacilityID: 3, Role: users.Student}
	pUsers  = []coreapi.ProviderUser{{Username: "jdoe", NameFirst: "Jane", NameLast: "Doe", Email: "j@x.org", ExternalUserID: "e1", ExternalUsername: "jd"}}
)

type fake struct {
	mu     sync.Mutex
	method string
	args   []any
	err    error
	ok     bool
}

func (f *fake) record(method string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.method, f.args = method, args
}

func (f *fake) asked() (string, []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.args
}

func (f *fake) Get(_ context.Context, id uint) (coreapi.User, error) {
	f.record("Get", id)
	return user, f.err
}

func (f *fake) Create(_ context.Context, u coreapi.NewUser) (coreapi.User, error) {
	f.record("Create", u)
	return user, f.err
}

func (f *fake) CreateLogin(_ context.Context, id uint) (string, error) {
	f.record("CreateLogin", id)
	return "temp-pass", f.err
}

func (f *fake) Delete(_ context.Context, id uint) error {
	f.record("Delete", id)
	return f.err
}

func (f *fake) Users(_ context.Context, providerID uint) ([]coreapi.ProviderUser, bool, error) {
	f.record("Users", providerID)
	return pUsers, f.ok, f.err
}

func (f *fake) MappingsChanged(providerID, userID uint) {
	f.record("MappingsChanged", providerID, userID)
}

type server struct {
	users coreapi.Users
	live  coreapi.LiveProviders
	f     *fake
	url   string
}

func start(t *testing.T) server {
	t.Helper()
	f := &fake{ok: true}
	mux := http.NewServeMux()
	mux.Handle("/internal/v1/core/", coreapi.NewHandler(f, f, token))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return server{
		users: coreapi.NewUsersClient(srv.URL, token, nil),
		live:  coreapi.NewLiveProvidersClient(srv.URL, token, nil),
		f:     f,
		url:   srv.URL,
	}
}

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := start(t)
	tests := []struct {
		name   string
		call   func() (any, error)
		want   any
		method string
		args   []any
	}{
		{"Get", func() (any, error) { return s.users.Get(ctx, 7) }, user, "Get", []any{uint(7)}},
		{"Create", func() (any, error) { return s.users.Create(ctx, newUser) }, user, "Create", []any{newUser}},
		{"CreateLogin", func() (any, error) { return s.users.CreateLogin(ctx, 7) }, "temp-pass", "CreateLogin", []any{uint(7)}},
		{"Delete", func() (any, error) { return nil, s.users.Delete(ctx, 7) }, nil, "Delete", []any{uint(7)}},
		{"Users", func() (any, error) {
			got, ok, err := s.live.Users(ctx, 2)
			return append([]any{ok}, got), err
		}, []any{true, pUsers}, "Users", []any{uint(2)}},
		{"MappingsChanged", func() (any, error) { s.live.MappingsChanged(2, 7); return nil, nil }, nil, "MappingsChanged", []any{uint(2), uint(7)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.call()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("answer = %#v, want %#v", got, tt.want)
			}
			method, args := s.f.asked()
			if method != tt.method || !reflect.DeepEqual(args, tt.args) {
				t.Errorf("asked %s%v, want %s%v", method, args, tt.method, tt.args)
			}
		})
	}
}

func TestUsersErrors(t *testing.T) {
	ctx := context.Background()
	s := start(t)

	s.f.err = errors.New("boom")
	if _, err := s.users.Get(ctx, 1); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Get error = %v, want boom", err)
	}
	if err := s.users.Delete(ctx, 1); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Delete error = %v, want boom", err)
	}

	s.f.err = coreapi.ErrUserNotFound
	if _, err := s.users.Get(ctx, 1); !errors.Is(err, coreapi.ErrUserNotFound) {
		t.Errorf("Get error = %v, want ErrUserNotFound", err)
	}
}

func TestLiveProvidersNotOK(t *testing.T) {
	s := start(t)
	s.f.ok = false
	_, ok, err := s.live.Users(context.Background(), 2)
	if err != nil || ok {
		t.Errorf("Users = ok %v, err %v; want false, nil", ok, err)
	}
}

func TestRejectsBadToken(t *testing.T) {
	ctx := context.Background()
	s := start(t)

	wrong := coreapi.NewUsersClient(s.url, "wrong", nil)
	if _, err := wrong.Get(ctx, 7); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("wrong token: error = %v, want 401", err)
	}
	if method, _ := s.f.asked(); method != "" {
		t.Errorf("wrong token reached %s", method)
	}

	for _, path := range []string{"/internal/v1/core/users/get", "/internal/v1/core/live-providers/users"} {
		resp, err := http.Post(s.url+path, "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s with no token: status = %d, want 401", path, resp.StatusCode)
		}
	}
	if method, _ := s.f.asked(); method != "" {
		t.Errorf("missing token reached %s", method)
	}
}

func TestWire(t *testing.T) {
	var gotPath, gotToken, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotToken, gotBody = r.URL.Path, r.Header.Get("Authorization"), string(b)
		_, _ = io.WriteString(w, `{"message":"","data":null}`)
	}))
	defer srv.Close()
	ctx := context.Background()
	u := coreapi.NewUsersClient(srv.URL, token, nil)
	l := coreapi.NewLiveProvidersClient(srv.URL, token, nil)

	tests := []struct {
		name string
		call func()
		path string
		body string
	}{
		{"Get", func() { _, _ = u.Get(ctx, 7) }, "/internal/v1/core/users/get", `{"id":7}`},
		{"Create", func() { _, _ = u.Create(ctx, newUser) }, "/internal/v1/core/users/create",
			`{"user":{"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"j@x.org","facility_id":3,"role":"student"}}`},
		{"CreateLogin", func() { _, _ = u.CreateLogin(ctx, 7) }, "/internal/v1/core/users/create-login", `{"id":7}`},
		{"Delete", func() { _ = u.Delete(ctx, 7) }, "/internal/v1/core/users/delete", `{"id":7}`},
		{"Users", func() { _, _, _ = l.Users(ctx, 2) }, "/internal/v1/core/live-providers/users", `{"provider_id":2}`},
		{"MappingsChanged", func() { l.MappingsChanged(2, 7) }, "/internal/v1/core/live-providers/mappings-changed", `{"provider_id":2,"user_id":7}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call()
			if gotPath != tt.path || gotBody != strings.TrimSpace(tt.body) || gotToken != "Bearer "+token {
				t.Errorf("sent %s %q token %q, want %s %q token %q", gotPath, gotBody, gotToken, tt.path, tt.body, "Bearer "+token)
			}
		})
	}

	t.Run("answers", func(t *testing.T) {
		f := &fake{ok: true}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/internal/v1/core/users/get", strings.NewReader(`{"id":7}`))
		req.Header.Set("Authorization", "Bearer "+token)
		coreapi.NewHandler(f, f, token).ServeHTTP(rec, req)
		want := `{"message":"","data":{"id":7,"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"j@x.org","facility_id":3}}`
		if got := strings.TrimSpace(rec.Body.String()); got != want {
			t.Errorf("Get answer = %s, want %s", got, want)
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/internal/v1/core/live-providers/users", strings.NewReader(`{"provider_id":2}`))
		req.Header.Set("Authorization", "Bearer "+token)
		coreapi.NewHandler(f, f, token).ServeHTTP(rec, req)
		want = `{"message":"","data":{"users":[{"username":"jdoe","name_first":"Jane","name_last":"Doe","email":"j@x.org","external_user_id":"e1","external_username":"jd"}],"ok":true}}`
		if got := strings.TrimSpace(rec.Body.String()); got != want {
			t.Errorf("Users answer = %s, want %s", got, want)
		}
	})
}

func TestUnmountedRouteIsNotUserNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NewServeMux())
	defer srv.Close()
	_, err := coreapi.NewUsersClient(srv.URL, token, nil).Get(context.Background(), 1)
	if err == nil || errors.Is(err, coreapi.ErrUserNotFound) {
		t.Errorf("Get on a bare 404 = %v, want an error that is not ErrUserNotFound", err)
	}
}

func TestCreateRoles(t *testing.T) {
	tests := []struct {
		role users.Role
		ok   bool
	}{
		{users.Student, true},
		{"", true},
		{users.SystemAdmin, false},
		{users.FacilityAdmin, false},
		{users.DepartmentAdmin, false},
		{"superuser", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			s := start(t)
			u := newUser
			u.Role = tt.role
			_, err := s.users.Create(context.Background(), u)
			method, _ := s.f.asked()
			if tt.ok && (err != nil || method != "Create") {
				t.Errorf("Create = %v, reached %q; want it to pass", err, method)
			}
			if !tt.ok && (err == nil || !strings.Contains(err.Error(), "400") || method != "") {
				t.Errorf("Create = %v, reached %q; want a 400 that never reaches Users", err, method)
			}
		})
	}
}
