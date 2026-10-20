package identity_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/features"
	"github.com/UnlockedLabs/unlocked-contracts/identity"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

func TestRoles(t *testing.T) {
	tests := []struct {
		role              users.Role
		isAdmin           bool
		canSwitchFacility bool
	}{
		{users.SystemAdmin, true, true},
		{users.DepartmentAdmin, true, true},
		{users.FacilityAdmin, true, false},
		{users.Student, false, false},
		{"", false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			c := identity.Claims{Role: tt.role}
			if got := c.IsAdmin(); got != tt.isAdmin {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.isAdmin)
			}
			if got := c.CanSwitchFacility(); got != tt.canSwitchFacility {
				t.Errorf("CanSwitchFacility() = %v, want %v", got, tt.canSwitchFacility)
			}
		})
	}
}

func TestHasFeature(t *testing.T) {
	c := identity.Claims{FeatureAccess: []features.Access{features.OpenContent, features.UploadVideo}}
	tests := []struct {
		name string
		fs   []features.Access
		want bool
	}{
		{"one that's on", []features.Access{features.OpenContent}, true},
		{"all of several that are on", []features.Access{features.OpenContent, features.UploadVideo}, true},
		{"one that's off", []features.Access{features.HelpfulLinks}, false},
		{"one on and one off", []features.Access{features.OpenContent, features.HelpfulLinks}, false},
		{"none asked for", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.HasFeature(tt.fs...); got != tt.want {
				t.Errorf("HasFeature(%v) = %v, want %v", tt.fs, got, tt.want)
			}
		})
	}
}

func TestContext(t *testing.T) {
	want := &identity.Claims{UserID: 7}
	got, ok := identity.FromContext(identity.NewContext(context.Background(), want))
	if !ok || got != want {
		t.Errorf("FromContext() = %v, %v, want %v, true", got, ok, want)
	}

	for name, ctx := range map[string]context.Context{
		"no claims":  context.Background(),
		"nil claims": identity.NewContext(context.Background(), nil),
	} {
		t.Run(name, func(t *testing.T) {
			if c, ok := identity.FromContext(ctx); ok || c != nil {
				t.Errorf("FromContext() = %v, %v, want nil, false", c, ok)
			}
		})
	}
}

// The JSON names are core's, so they can't change without breaking callers.
func TestClaimsJSON(t *testing.T) {
	c := identity.Claims{
		UserID: 1, Username: "jdoe", Role: users.Student, FacilityID: 2, FacilityName: "North",
		FeatureAccess: []features.Access{features.OpenContent}, SessionID: "s", TimeZone: "America/Chicago",
	}
	got, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"user_id":1,"username":"jdoe","role":"student","facility_id":2,"facility_name":"North",` +
		`"feature_access":["open_content"],"session_id":"s","timezone":"America/Chicago"}`
	if string(got) != want {
		t.Errorf("json = %s\nwant   %s", got, want)
	}
}
