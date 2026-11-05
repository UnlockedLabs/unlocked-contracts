package programs

import (
	"time"

	"github.com/UnlockedLabs/unlocked-contracts/coreapi"
)

// The paths Handler serves and Client calls, one per question. Each is a POST
// with the question's arguments as JSON. Only reachable from inside the
// cluster: the ingress never routes /internal.
const (
	basePath = "/internal/v1/programs/"

	pathEnrolledUserIDs           = basePath + "enrolled-user-ids"
	pathEligibleUserIDs           = basePath + "eligible-user-ids"
	pathTransferConflicts         = basePath + "transfer-conflicts"
	pathEndEnrollmentsForTransfer = basePath + "end-enrollments-for-transfer"
	pathWithdrawEnrollments       = basePath + "withdraw-enrollments"
	pathResidentPrograms          = basePath + "resident-programs"
	pathFacilityPrograms          = basePath + "facility-programs"
	pathClassHistoryCreatedAt     = basePath + "class-history-created-at"
	pathEngagementOverview        = basePath + "engagement-overview"
	pathSecondProgramEnrollment   = basePath + "second-program-enrollment"
	pathCompletionMatrix          = basePath + "completion-matrix"
	pathLoadDistribution          = basePath + "load-distribution"
	pathEnrollmentByProgramType   = basePath + "enrollment-by-program-type"
	pathClassMetrics              = basePath + "class-metrics"
	pathFacilityHealth            = basePath + "facility-health"
	pathLiveUsers                 = basePath + "live-users"
	pathMappingsChanged           = basePath + "mappings-changed"
)

// The arguments each question sends.
type (
	classRequest struct {
		ClassID uint `json:"class_id"`
	}
	userRequest struct {
		UserID     uint `json:"user_id"`
		FacilityID uint `json:"facility_id,omitempty"`
	}
	facilitiesRequest struct {
		FacilityIDs []uint `json:"facility_ids"`
	}
	idsRequest struct {
		IDs []uint `json:"ids"`
	}
	facilityRequest struct {
		FacilityID *uint `json:"facility_id,omitempty"`
	}
	overviewRequest struct {
		Start      *time.Time `json:"start,omitempty"`
		End        *time.Time `json:"end,omitempty"`
		FacilityID *uint      `json:"facility_id,omitempty"`
	}
	healthRequest struct {
		FacilityID *uint  `json:"facility_id,omitempty"`
		Days       int    `json:"days"`
		TZ         string `json:"tz"`
	}
	providerRequest struct {
		ProviderID uint `json:"provider_id"`
	}
	mappingsRequest struct {
		ProviderID uint `json:"provider_id"`
		UserID     uint `json:"user_id"`
	}
	liveUsersAnswer struct {
		Users []coreapi.ProviderUser `json:"users"`
		OK    bool                   `json:"ok"`
	}
)
