// Package identity says who is logged in. Core checks the login session and
// builds the Claims; a feature module reads them from the request's context.
// A product service that gets requests straight from the ingress checks the
// session itself with a SessionChecker, which asks core for the same Claims.
package identity

import (
	"context"
	"slices"

	"github.com/UnlockedLabs/unlocked-contracts/features"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

// Claims describe the logged-in user. The JSON names match the claims core
// already sends, so the same shape works over HTTP later.
type Claims struct {
	UserID        uint              `json:"user_id"`
	Username      string            `json:"username"`
	Role          users.Role        `json:"role"`
	FacilityID    uint              `json:"facility_id"`
	FacilityName  string            `json:"facility_name"`
	FeatureAccess []features.Access `json:"feature_access"`
	SessionID     string            `json:"session_id"`
	TimeZone      string            `json:"timezone"`
}

// IsAdmin reports whether the user has any admin role.
func (c *Claims) IsAdmin() bool {
	return slices.Contains(users.AdminRoles, c.Role)
}

// CanSwitchFacility reports whether the user may view facilities other than
// their own: system and department admins can, facility admins and residents
// can't.
func (c *Claims) CanSwitchFacility() bool {
	return c.Role == users.SystemAdmin || c.Role == users.DepartmentAdmin
}

// HasFeature reports whether every one of fs is on for the user's facility.
func (c *Claims) HasFeature(fs ...features.Access) bool {
	for _, f := range fs {
		if !slices.Contains(c.FeatureAccess, f) {
			return false
		}
	}
	return true
}

type contextKey struct{}

// NewContext returns a copy of ctx that carries c.
func NewContext(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, c)
}

// FromContext returns the Claims in ctx, and false if there are none.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(contextKey{}).(*Claims)
	return c, ok && c != nil
}
