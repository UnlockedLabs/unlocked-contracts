package permissions_test

import (
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/permissions"
)

// The screen names are stored in the database, so they can't change.
func TestScreens(t *testing.T) {
	tests := []struct {
		screen permissions.Screen
		want   string
	}{
		{permissions.Programs, "programs"},
		{permissions.Classes, "program_classes"},
		{permissions.Schedule, "program_schedule"},
		{permissions.Enrollments, "program_enrollments"},
		{permissions.Attendance, "program_attendance"},
		{permissions.Reports, "program_reports"},
	}
	if len(permissions.All) != len(tests) {
		t.Errorf("All has %d screens, want %d", len(permissions.All), len(tests))
	}
	for i, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.screen) != tt.want {
				t.Errorf("got %q, want %q", tt.screen, tt.want)
			}
			if i < len(permissions.All) && permissions.All[i] != tt.screen {
				t.Errorf("All[%d] = %q, want %q", i, permissions.All[i], tt.screen)
			}
		})
	}
}

func TestGrantAllows(t *testing.T) {
	tests := []struct {
		name   string
		grant  permissions.Grant
		action permissions.Action
		want   bool
	}{
		{"view needs no flag", permissions.Grant{}, permissions.View, true},
		{"add", permissions.Grant{Add: true}, permissions.Add, true},
		{"add not granted", permissions.Grant{Edit: true, Delete: true}, permissions.Add, false},
		{"edit", permissions.Grant{Edit: true}, permissions.Edit, true},
		{"edit not granted", permissions.Grant{Add: true, Delete: true}, permissions.Edit, false},
		{"delete", permissions.Grant{Delete: true}, permissions.Delete, true},
		{"delete not granted", permissions.Grant{Add: true, Edit: true}, permissions.Delete, false},
		{"unknown action", permissions.Grant{Add: true, Edit: true, Delete: true}, "export", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.grant.Allows(tt.action); got != tt.want {
				t.Errorf("Allows(%q) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}
