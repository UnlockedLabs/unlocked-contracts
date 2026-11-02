// Package opencontent is what core asks Open Content, which runs as its own
// service (EN-205): the OpenContent interface and its types, a Client that
// core uses to ask over HTTP, and a Handler that Open Content serves the
// answers with. It also names the NATS message Open Content sends to core's
// websocket.
//
// The JSON names of ContentUsage, RepeatEngagement, CategoryViews and
// ContentVisits match what core sends its browsers, so core can pass them on
// unchanged.
package opencontent

import (
	"context"
	"time"
)

// OpenContent answers core's questions about libraries, videos and reading
// activity.
type OpenContent interface {
	// EndActivity records that the user stopped viewing the content of an
	// activity. Core calls it when the user's tab says so or closes. It logs a
	// failure instead of returning it.
	EndActivity(ctx context.Context, activityID int64)

	// TopLibraries is a resident's five most-read libraries, with the
	// visibility at facilityID.
	TopLibraries(ctx context.Context, userID, facilityID uint) ([]ContentUsage, error)
	// RecentVideos is the five videos a resident watched most recently.
	RecentVideos(ctx context.Context, userID uint) ([]ContentUsage, error)
	// ResourcesAccessed counts the different libraries, videos and links a
	// resident has opened.
	ResourcesAccessed(ctx context.Context, userID uint) (int64, error)
	// ActivityEngagement is a resident's reading time, for their profile.
	ActivityEngagement(ctx context.Context, userID uint) (ActivityEngagement, error)

	// FacilityContent is the Open Content data each of facilityIDs holds, for
	// core's facility list and facility delete check. A facility that holds
	// none is left out.
	FacilityContent(ctx context.Context, facilityIDs []uint) (map[uint]FacilityContent, error)

	// The Knowledge Center numbers, for the activity between start and end
	// (all of it when either is nil) at facilityID (every facility when nil).
	// The change values compare with the window of the same length before
	// start.
	Interactions(ctx context.Context, w Window) (Interactions, error)
	AvgSessionMinutes(ctx context.Context, w Window) (float64, error)
	RepeatEngagement(ctx context.Context, w Window) (RepeatEngagement, error)
	LibraryViewsByCategory(ctx context.Context, w Window) ([]CategoryViews, error)
	TopLibraryVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error)
	TopVideoVisits(ctx context.Context, w Window, limit int) ([]ContentVisits, error)

	// ResetDemoActivity deletes demo reading activity from before the given
	// day.
	ResetDemoActivity(ctx context.Context, before time.Time) error
	// SeedDemoActivity adds six months of made-up reading activity for each
	// user, at facilityID.
	SeedDemoActivity(ctx context.Context, facilityID uint, userIDs []uint) error
}

// ContentUsage is one library or video a resident used, and for how long.
type ContentUsage struct {
	Title                 string `json:"title"`
	Url                   string `json:"url"`
	ThumbnailUrl          string `json:"thumbnail_url"`
	Description           string `json:"description,omitempty"`
	VisibilityStatus      *bool  `json:"visibility_status"`
	OpenContentProviderId uint   `json:"open_content_provider_id"`
	ContentId             uint   `json:"content_id"`
	ContentType           string `json:"content_type"`
	ProviderName          string `json:"provider_name,omitempty"`
	ChannelTitle          string `json:"channel_title,omitempty"`

	IsFeatured   bool    `json:"is_featured"`
	TotalHours   float64 `json:"total_hours"`
	TotalMinutes float64 `json:"total_minutes"`
}

// ActivityEngagement totals a resident's reading time.
type ActivityEngagement struct {
	// TotalActiveDaysMonthly is the days with any reading activity.
	TotalActiveDaysMonthly int64 `json:"total_active_days_monthly"`
	// TotalHoursActiveMonthly is the average activity's length, in hours.
	TotalHoursActiveMonthly float64 `json:"total_hours_active_monthly"`
	// TotalHoursActiveWeekly and TotalMinutesActiveWeekly are all the
	// activity's time, divided by four.
	TotalHoursActiveWeekly   float64 `json:"total_hours_active_weekly"`
	TotalMinutesActiveWeekly float64 `json:"total_minutes_active_weekly"`
	// TotalHoursEngaged and TotalMinutesEngaged are this week's activity time.
	TotalHoursEngaged   float64 `json:"total_hours_engaged"`
	TotalMinutesEngaged float64 `json:"total_minutes_engaged"`
}

// FacilityContent counts the Open Content rows recorded at a facility.
type FacilityContent struct {
	Activities         int64 `json:"activities"`
	Favorites          int64 `json:"favorites"`
	VisibilityStatuses int64 `json:"visibility_statuses"`
}

// Window is the activity a Knowledge Center number counts.
type Window struct {
	Start      *time.Time `json:"start,omitempty"`
	End        *time.Time `json:"end,omitempty"`
	FacilityID *uint      `json:"facility_id,omitempty"`
}

// Interactions counts reading activity in a window.
type Interactions struct {
	Total  int64 `json:"total"`
	Unique int64 `json:"unique"`
	// Change is the percent change in Total from the window before.
	Change int `json:"change"`
}

// RepeatEngagement counts residents by how often they read in a window.
type RepeatEngagement struct {
	Once      int64 `json:"once"`
	TwoToFour int64 `json:"two_to_four"`
	FivePlus  int64 `json:"five_plus"`
}

// CategoryViews counts library visits by category.
type CategoryViews struct {
	Category string `json:"category"`
	Views    int64  `json:"views"`
}

// ContentVisits counts visits to one library or video, and the percent
// change from the window before.
type ContentVisits struct {
	Title  string `json:"title"`
	Visits int64  `json:"visits"`
	Change int    `json:"change"`
}
