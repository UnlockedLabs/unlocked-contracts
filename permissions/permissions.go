// Package permissions names the screens an admin can grant access to and what
// a group may do on each one. Core stores the grants; a feature module checks
// them with identity.Claims.Can.
package permissions

// Screen is a part of the app an admin can grant access to. The values are
// stored in the database, so they never change.
type Screen string

// Program Tracking screens.
const (
	Programs    Screen = "programs"
	Classes     Screen = "program_classes"
	Schedule    Screen = "program_schedule"
	Enrollments Screen = "program_enrollments"
	Attendance  Screen = "program_attendance"
	Reports     Screen = "program_reports"
)

// All lists every screen.
var All = []Screen{Programs, Classes, Schedule, Enrollments, Attendance, Reports}

// Action is something a user does on a screen.
type Action string

// Actions.
const (
	View   Action = "view"
	Add    Action = "add"
	Edit   Action = "edit"
	Delete Action = "delete"
)

// Grant is what a user may do on one screen. Having a Grant for a screen at
// all means they may view it.
type Grant struct {
	Add    bool `json:"add"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// Allows reports whether g lets its holder take action a. An unknown action
// is never allowed.
func (g Grant) Allows(a Action) bool {
	switch a {
	case View:
		return true
	case Add:
		return g.Add
	case Edit:
		return g.Edit
	case Delete:
		return g.Delete
	default:
		return false
	}
}
