package providerplatforms

import (
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
)

// The paths Handler serves and Client calls, one per question. Each is a POST
// with the question's arguments as JSON. Only reachable from inside the
// cluster: the ingress never routes /internal.
const (
	basePath = "/internal/v1/provider-platforms/"

	pathProvider               = basePath + "provider"
	pathActiveProviders        = basePath + "active-providers"
	pathUserMappings           = basePath + "user-mappings"
	pathMappedUserIDs          = basePath + "mapped-user-ids"
	pathDeleteUserMappings     = basePath + "delete-user-mappings"
	pathMappedUsers            = basePath + "mapped-users"
	pathCountMapped            = basePath + "count-mapped"
	pathCountMappedAtFacility  = basePath + "count-mapped-at-facility"
	pathCountMappedPerFacility = basePath + "count-mapped-per-facility"
	pathAddUserToProviders     = basePath + "add-user-to-providers"
	pathAddUserToKolibri       = basePath + "add-user-to-kolibri"
	pathSetupKolibri           = basePath + "setup-kolibri"
	pathLearningStats          = basePath + "learning-stats"
	pathLastDemoSeed           = basePath + "last-demo-seed"
	pathResetDemoLearning      = basePath + "reset-demo-learning"
	pathSeedDemoLearning       = basePath + "seed-demo-learning"
)

// The arguments each question sends.
type (
	noRequest struct{}
	idRequest struct {
		ID uint `json:"id"`
	}
	userRequest struct {
		UserID uint `json:"user_id"`
	}
	providerRequest struct {
		ProviderID uint `json:"provider_id"`
	}
	mappedRequest struct {
		ProviderID      uint     `json:"provider_id"`
		ExternalUserIDs []string `json:"external_user_ids"`
		FacilityID      uint     `json:"facility_id,omitempty"`
	}
	addToProvidersRequest struct {
		User        coreapi.User `json:"user"`
		ProviderIDs []uint       `json:"provider_ids"`
	}
	addToKolibriRequest struct {
		User coreapi.User `json:"user"`
	}
	statsRequest struct {
		FacilityID *uint `json:"facility_id,omitempty"`
	}
	resetDemoRequest struct {
		Through time.Time `json:"through"`
	}
	seedDemoRequest struct {
		UserIDs []uint `json:"user_ids"`
	}
	lastDemoSeedAnswer struct {
		Last  time.Time `json:"last"`
		Found bool      `json:"found"`
	}
)
