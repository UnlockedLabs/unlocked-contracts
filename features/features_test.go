package features_test

import (
	"slices"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/features"
)

func TestValues(t *testing.T) {
	tests := []struct {
		feature features.Access
		want    string
	}{
		{features.OpenContent, "open_content"},
		{features.ProviderPlatforms, "provider_platforms"},
		{features.ProgramManagement, "program_management"},
		{features.LearningRecord, "learning_record"},
		{features.AITutor, "ai_tutor"},
		{features.RequestContent, "request_content"},
		{features.HelpfulLinks, "helpful_links"},
		{features.UploadVideo, "upload_video"},
		{features.ResidentPrograms, "resident_programs"},
	}
	if len(features.All) != len(tests) {
		t.Fatalf("All has %d features, want %d", len(features.All), len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.feature) != tt.want {
				t.Errorf("got %q, want %q", tt.feature, tt.want)
			}
			if !slices.Contains(features.All, tt.feature) {
				t.Errorf("%q is missing from All", tt.feature)
			}
		})
	}
}
