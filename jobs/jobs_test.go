package jobs_test

import (
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/jobs"
)

func TestJobs(t *testing.T) {
	tests := []struct {
		job       jobs.Type
		pubName   string
		isVideo   bool
		isLibrary bool
	}{
		{jobs.GetMilestones, "tasks.get_milestones", false, false},
		{jobs.GetCourses, "tasks.get_courses", false, false},
		{jobs.GetActivity, "tasks.get_activity", false, false},
		{jobs.ScrapeKiwix, "tasks.scrape_kiwix", false, true},
		{jobs.RetryVideoDownloads, "tasks.retry_video_downloads", true, false},
		{jobs.RetryManualDownload, "tasks.retry_manual_download", false, false},
		{jobs.SyncVideoMetadata, "tasks.sync_video_metadata", true, false},
		{jobs.AddVideos, "tasks.add_videos", false, false},
		{jobs.ActivateScheduledClasses, "tasks.activate_scheduled_classes", false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.job), func(t *testing.T) {
			if got := tt.job.PubName(); got != tt.pubName {
				t.Errorf("PubName() = %q, want %q", got, tt.pubName)
			}
			if got := tt.job.IsVideoJob(); got != tt.isVideo {
				t.Errorf("IsVideoJob() = %v, want %v", got, tt.isVideo)
			}
			if got := tt.job.IsLibraryJob(); got != tt.isLibrary {
				t.Errorf("IsLibraryJob() = %v, want %v", got, tt.isLibrary)
			}
		})
	}
}
