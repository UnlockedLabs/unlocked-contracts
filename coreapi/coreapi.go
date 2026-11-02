// Package coreapi is what a product service asks core: the user accounts it
// adds or looks up (Users) and the live LMS data Programs holds
// (LiveProviders). Each has a Client for the service and a Handler that core
// serves, both under /internal/v1/core/, so they always agree.
//
// Every call carries a shared service token, which the Handler checks. The
// path is only reachable from inside the cluster: the ingress never routes
// /internal.
package coreapi

import (
	"context"

	"github.com/UnlockedLabs/unlocked-contracts/internal/rpc"
	"github.com/UnlockedLabs/unlocked-contracts/users"
)

// ErrUserNotFound is what Users.Get returns for a missing user. A core that
// implements Users returns an error wrapping it.
var ErrUserNotFound = rpc.ErrNotFound

// Users is core's user accounts, for a service that adds or looks up users.
type Users interface {
	// Get returns the user with id. When there is none, the error matches
	// ErrUserNotFound.
	Get(ctx context.Context, id uint) (User, error)
	// Create adds a user. The user can't log in until CreateLogin.
	Create(ctx context.Context, u NewUser) (User, error)
	// CreateLogin gives a user a Kratos login with a new temporary password,
	// which they must change when they first log in. It returns the password.
	CreateLogin(ctx context.Context, userID uint) (string, error)
	// Delete removes a user and their links to provider platform accounts,
	// such as to undo a Create. It doesn't remove their Kratos login.
	Delete(ctx context.Context, id uint) error
}

// LiveProviders is what Provider Platforms needs from Programs' Canvas and
// Essential Education code.
type LiveProviders interface {
	// Users is every user a provider's own API lists. ok is false for a
	// provider that can't list them; its users come from Provider Platforms'
	// worker.
	Users(ctx context.Context, providerID uint) (users []ProviderUser, ok bool, err error)
	// MappingsChanged tells Programs that a user's account at a provider was
	// linked or unlinked, so it can clear its cached Canvas programs. It logs
	// a failure instead of returning it.
	MappingsChanged(providerID, userID uint)
}

// User is what a service sees of a user.
type User struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	NameFirst  string `json:"name_first"`
	NameLast   string `json:"name_last"`
	Email      string `json:"email"`
	FacilityID uint   `json:"facility_id"`
}

// NewUser is a user for Users.Create to add.
type NewUser struct {
	Username   string     `json:"username"`
	NameFirst  string     `json:"name_first"`
	NameLast   string     `json:"name_last"`
	Email      string     `json:"email"`
	FacilityID uint       `json:"facility_id"`
	Role       users.Role `json:"role"`
}

// ProviderUser is a user's account at a provider platform.
type ProviderUser struct {
	Username         string `json:"username"`
	NameFirst        string `json:"name_first"`
	NameLast         string `json:"name_last"`
	Email            string `json:"email"`
	ExternalUserID   string `json:"external_user_id"`
	ExternalUsername string `json:"external_username"`
}
