// Package features names the features that can be turned on or off per facility.
package features

// Access is a feature name. The values are stored in the database, so they
// never change.
type Access string

// Top-level features.
const (
	OpenContent       Access = "open_content"
	ProviderPlatforms Access = "provider_platforms"
	ProgramManagement Access = "program_management"
	LearningRecord    Access = "learning_record"
	AITutor           Access = "ai_tutor"
)

// Page-level features, each gated by a top-level feature.
const (
	RequestContent   Access = "request_content"
	HelpfulLinks     Access = "helpful_links"
	UploadVideo      Access = "upload_video"
	ResidentPrograms Access = "resident_programs"
)

// All lists every feature.
var All = []Access{
	OpenContent, ProviderPlatforms, ProgramManagement, LearningRecord, AITutor,
	RequestContent, HelpfulLinks, UploadVideo, ResidentPrograms,
}
