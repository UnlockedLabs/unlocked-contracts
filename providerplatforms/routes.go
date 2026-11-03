package providerplatforms

import "github.com/UnlockedLabs/unlocked-contracts/features"

// Access is who may call a route.
type Access string

const (
	// Admin is any admin: facility, department or system.
	Admin Access = "admin"
	// AnyUser is any logged-in user.
	AnyUser Access = "any_user"
)

// Route is one HTTP route Provider Platforms serves under core's /api.
type Route struct {
	// Pattern is a Go 1.22 mux pattern, such as "GET /api/courses/{id}".
	Pattern string
	Access  Access
	// UserParam names the path value that holds the user the route is about,
	// or is empty. Only a caller who may see that user (an admin, and with
	// AnyUser the user themself) may call the route.
	UserParam string
	// Feature must be enabled at the caller's facility.
	Feature features.Access
}

func route(pattern string, access Access, userParam string) Route {
	return Route{Pattern: pattern, Access: access, UserParam: userParam, Feature: features.ProviderPlatforms}
}

// Routes is every route core had for Provider Platforms (EN-193). Core's proxy
// enforces Access, UserParam and Feature before it forwards a request to the
// service.
var Routes = []Route{
	route("GET /api/provider-platforms", Admin, ""),
	route("GET /api/provider-platforms/{id}", Admin, ""),
	route("POST /api/provider-platforms", Admin, ""),
	route("GET /api/provider-platforms/callback", Admin, ""),
	route("GET /api/provider-platforms/{id}/refresh", Admin, ""),
	route("PATCH /api/provider-platforms/{id}", Admin, ""),
	route("DELETE /api/provider-platforms/{id}", Admin, ""),
	route("POST /api/provider-platforms/{id}/map-user/{user_id}", Admin, "user_id"),
	route("POST /api/provider-platforms/{id}/users/import", Admin, ""),
	route("POST /api/provider-platforms/{id}/create-user/{user_id}", Admin, ""),
	route("GET /api/users/{id}/logins", Admin, ""),
	route("POST /api/users/{id}/logins", Admin, ""),
	route("POST /api/provider-platforms/{id}/user-accounts/{user_id}", Admin, ""),
	route("DELETE /api/users/{userId}/logins/{providerId}", Admin, ""),
	route("GET /api/provider-platforms/{id}/mapped-users", Admin, ""),
	route("GET /api/actions/provider-platforms/{id}/get-users", Admin, ""),
	route("POST /api/actions/provider-platforms/{id}/import-users", Admin, ""),
	route("GET /api/actions/provider-platforms/{id}/match-users", Admin, ""),
	route("POST /api/actions/provider-platforms/{id}/apply-matches", Admin, ""),
	route("GET /api/oidc/clients", Admin, ""),
	route("POST /api/oidc/clients", Admin, ""),
	route("GET /api/oidc/clients/{id}", Admin, ""),
	route("POST /api/oidc/clients/{id}/recreate", Admin, ""),
	route("GET /api/courses", Admin, ""),
	route("GET /api/courses/{id}", AnyUser, ""),
	route("GET /api/courses/{id}/activity", Admin, ""),
	route("GET /api/users/{id}/outcomes", AnyUser, ""),
	route("GET /api/users/{id}/daily-activity", AnyUser, "id"),
	route("GET /api/users/{id}/catalog", AnyUser, "id"),
	route("GET /api/users/{id}/courses", AnyUser, "id"),
}
