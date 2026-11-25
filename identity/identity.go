// Package identity says who is logged in. Core checks the login session and
// builds the Claims; a feature module reads them from the request's context.
// A product service that gets requests straight from the ingress checks the
// session itself with a SessionChecker, which asks core for the same Claims.
package identity

import (
	"context"
	"slices"

	"github.com/UnlockedLabs/unlocked-contracts/features"
	"github.com/UnlockedLabs/unlocked-contracts/permissions"
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
	// Permissions is what the user's groups let them do, by screen. A screen
	// that isn't here can't be viewed.
	Permissions map[permissions.Screen]permissions.Grant `json:"permissions"`
	// EnforcedScreens lists the screens whose permissions are switched on.
	// On every other screen, permissions don't decide anything yet.
	EnforcedScreens []permissions.Screen `json:"enforced_screens"`
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

// Can reports whether the user's permissions let them take action a on screen
// s. It only ever narrows access, so a route still runs its other checks too.
// It's true when the screen's permissions are switched off, and for system
// admins and residents, whom permissions don't govern.
func (c *Claims) Can(s permissions.Screen, a permissions.Action) bool {
	if c.Role == users.SystemAdmin || !c.IsAdmin() || !slices.Contains(c.EnforcedScreens, s) {
		return true
	}
	g, ok := c.Permissions[s]
	return ok && g.Allows(a)
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
