package identity_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/identity"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

func TestClaimsHeaderRoundTrip(t *testing.T) {
	want := identity.Claims{UserID: 7, Username: "jdoe", Role: users.FacilityAdmin, FacilityID: 2}
	out, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	if err := identity.SetClaimsHeader(out, &want); err != nil {
		t.Fatal(err)
	}
	if out.Header.Get(identity.ClaimsHeader) == "" {
		t.Fatal("header not set")
	}
	got, ok := identity.ClaimsFromRequest(out)
	if !ok || got.UserID != want.UserID || got.Username != want.Username || got.Role != want.Role || got.FacilityID != want.FacilityID {
		t.Errorf("ClaimsFromRequest() = %v, %v, want %v, true", got, ok, want)
	}
}

func TestClaimsFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		header *string
		wantOK bool
	}{
		{"missing header", nil, false},
		{"not JSON", ptr("{"), false},
		{"empty object", ptr("{}"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != nil {
				r.Header.Set(identity.ClaimsHeader, *tt.header)
			}
			c, ok := identity.ClaimsFromRequest(r)
			if ok != tt.wantOK || (c != nil) != tt.wantOK {
				t.Errorf("ClaimsFromRequest() = %v, %v, want ok=%v", c, ok, tt.wantOK)
			}
		})
	}
}

func ptr(s string) *string { return &s }
