package providerplatforms_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/features"
	"github.com/UnlockedLabs/unlocked-contracts/providerplatforms"
)

const (
	admin   = providerplatforms.Admin
	anyUser = providerplatforms.AnyUser
)

func route(pattern string, access providerplatforms.Access, userParam string) providerplatforms.Route {
	return providerplatforms.Route{Pattern: pattern, Access: access, UserParam: userParam, Feature: features.ProviderPlatforms}
}

// TestRoutes pins every route core's module had, so a change shows as a diff.
func TestRoutes(t *testing.T) {
	want := []providerplatforms.Route{
		route("GET /api/provider-platforms", admin, ""),
		route("GET /api/provider-platforms/{id}", admin, ""),
		route("POST /api/provider-platforms", admin, ""),
		route("GET /api/provider-platforms/callback", admin, ""),
		route("GET /api/provider-platforms/{id}/refresh", admin, ""),
		route("PATCH /api/provider-platforms/{id}", admin, ""),
		route("DELETE /api/provider-platforms/{id}", admin, ""),
		route("POST /api/provider-platforms/{id}/map-user/{user_id}", admin, "user_id"),
		route("POST /api/provider-platforms/{id}/users/import", admin, ""),
		route("POST /api/provider-platforms/{id}/create-user/{user_id}", admin, ""),
		route("GET /api/users/{id}/logins", admin, ""),
		route("POST /api/users/{id}/logins", admin, ""),
		route("POST /api/provider-platforms/{id}/user-accounts/{user_id}", admin, ""),
		route("DELETE /api/users/{userId}/logins/{providerId}", admin, ""),
		route("GET /api/provider-platforms/{id}/mapped-users", admin, ""),
		route("GET /api/actions/provider-platforms/{id}/get-users", admin, ""),
		route("POST /api/actions/provider-platforms/{id}/import-users", admin, ""),
		route("GET /api/actions/provider-platforms/{id}/match-users", admin, ""),
		route("POST /api/actions/provider-platforms/{id}/apply-matches", admin, ""),
		route("GET /api/oidc/clients", admin, ""),
		route("POST /api/oidc/clients", admin, ""),
		route("GET /api/oidc/clients/{id}", admin, ""),
		route("POST /api/oidc/clients/{id}/recreate", admin, ""),
		route("GET /api/courses", admin, ""),
		route("GET /api/courses/{id}", anyUser, ""),
		route("GET /api/courses/{id}/activity", admin, ""),
		route("GET /api/users/{id}/outcomes", anyUser, ""),
		route("GET /api/users/{id}/daily-activity", anyUser, "id"),
		route("GET /api/users/{id}/catalog", anyUser, "id"),
		route("GET /api/users/{id}/courses", anyUser, "id"),
	}
	got := providerplatforms.Routes
	if len(got) != len(want) {
		t.Fatalf("%d routes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestRoutesParse registers every pattern on one mux, which panics on a
// pattern it can't parse or one that is registered twice.
func TestRoutesParse(t *testing.T) {
	mux := http.NewServeMux()
	for _, r := range providerplatforms.Routes {
		mux.HandleFunc(r.Pattern, func(http.ResponseWriter, *http.Request) {})
		if r.UserParam != "" && !strings.Contains(r.Pattern, "{"+r.UserParam+"}") {
			t.Errorf("%s: user param %q is not in the pattern", r.Pattern, r.UserParam)
		}
	}
}
