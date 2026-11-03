package opencontent

import "time"

// The paths Handler serves and Client calls, one per question. Each is a POST
// with the question's arguments as JSON. Only reachable from inside the
// cluster: the ingress never routes /internal.
const (
	basePath = "/internal/v1/open-content/"

	pathEndActivity            = basePath + "end-activity"
	pathTopLibraries           = basePath + "top-libraries"
	pathRecentVideos           = basePath + "recent-videos"
	pathResourcesAccessed      = basePath + "resources-accessed"
	pathActivityEngagement     = basePath + "activity-engagement"
	pathFacilityContent        = basePath + "facility-content"
	pathInteractions           = basePath + "interactions"
	pathAvgSessionMinutes      = basePath + "avg-session-minutes"
	pathRepeatEngagement       = basePath + "repeat-engagement"
	pathLibraryViewsByCategory = basePath + "library-views-by-category"
	pathTopLibraryVisits       = basePath + "top-library-visits"
	pathTopVideoVisits         = basePath + "top-video-visits"
	pathResetDemoActivity      = basePath + "reset-demo-activity"
	pathSeedDemoActivity       = basePath + "seed-demo-activity"
)

// The arguments each question sends.
type (
	activityRequest struct {
		ActivityID int64 `json:"activity_id"`
	}
	userRequest struct {
		UserID     uint `json:"user_id"`
		FacilityID uint `json:"facility_id,omitempty"`
	}
	facilitiesRequest struct {
		FacilityIDs []uint `json:"facility_ids"`
	}
	windowRequest struct {
		Window Window `json:"window"`
		Limit  int    `json:"limit,omitempty"`
	}
	resetDemoRequest struct {
		Before time.Time `json:"before"`
	}
	seedDemoRequest struct {
		FacilityID uint   `json:"facility_id"`
		UserIDs    []uint `json:"user_ids"`
	}
)
