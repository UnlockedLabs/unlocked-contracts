package users_test

import (
	"slices"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/users"
)

func TestRoles(t *testing.T) {
	tests := []struct {
		role    users.Role
		want    string
		isAdmin bool
	}{
		{users.SystemAdmin, "system_admin", true},
		{users.FacilityAdmin, "facility_admin", true},
		{users.DepartmentAdmin, "department_admin", true},
		{users.Student, "student", false},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.role) != tt.want {
				t.Errorf("got %q, want %q", tt.role, tt.want)
			}
			if got := slices.Contains(users.AdminRoles, tt.role); got != tt.isAdmin {
				t.Errorf("in AdminRoles = %v, want %v", got, tt.isAdmin)
			}
		})
	}
}
