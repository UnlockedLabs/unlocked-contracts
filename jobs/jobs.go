// Package jobs names the background jobs core schedules and workers run over NATS.
package jobs

// Type is a job name. The values are stored in the database and are part of
// each job's NATS subject, so they never change.
type Type string

// Provider platform jobs.
const (
	GetMilestones Type = "get_milestones"
	GetCourses    Type = "get_courses"
	GetActivity   Type = "get_activity"
)

// Open content jobs.
const (
	ScrapeKiwix         Type = "scrape_kiwix"
	RetryVideoDownloads Type = "retry_video_downloads"
	RetryManualDownload Type = "retry_manual_download"
	SyncVideoMetadata   Type = "sync_video_metadata"
	AddVideos           Type = "add_videos"
)

// System jobs.
const (
	ActivateScheduledClasses Type = "activate_scheduled_classes"
)

// PubName returns the NATS subject the job is published on.
func (t Type) PubName() string {
	return "tasks." + string(t)
}

// IsVideoJob reports whether the job works on videos.
func (t Type) IsVideoJob() bool {
	switch t {
	case RetryVideoDownloads, SyncVideoMetadata:
		return true
	}
	return false
}

// IsLibraryJob reports whether the job works on libraries.
func (t Type) IsLibraryJob() bool {
	switch t {
	case ScrapeKiwix:
		return true
	}
	return false
}
