package coreapi

// The paths Handler serves and the clients call, one per method. Each is a
// POST with the method's arguments as JSON.
const (
	basePath = "/internal/v1/core/"

	pathGet             = basePath + "users/get"
	pathCreate          = basePath + "users/create"
	pathCreateLogin     = basePath + "users/create-login"
	pathDelete          = basePath + "users/delete"
	pathLiveUsers       = basePath + "live-providers/users"
	pathMappingsChanged = basePath + "live-providers/mappings-changed"
)

// The arguments each method sends.
type (
	idRequest struct {
		ID uint `json:"id"`
	}
	createRequest struct {
		User NewUser `json:"user"`
	}
	providerRequest struct {
		ProviderID uint `json:"provider_id"`
	}
	mappingsRequest struct {
		ProviderID uint `json:"provider_id"`
		UserID     uint `json:"user_id"`
	}
	liveUsersAnswer struct {
		Users []ProviderUser `json:"users"`
		OK    bool           `json:"ok"`
	}
)
