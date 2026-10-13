package types_test

import (
	"slices"
	"testing"

	"github.com/UnlockedLabs/unlocked-contracts/types"
)

func TestUserRoles(t *testing.T) {
	tests := []struct {
		role    types.UserRole
		want    string
		isAdmin bool
	}{
		{types.SystemAdmin, "system_admin", true},
		{types.FacilityAdmin, "facility_admin", true},
		{types.DepartmentAdmin, "department_admin", true},
		{types.Student, "student", false},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.role) != tt.want {
				t.Errorf("got %q, want %q", tt.role, tt.want)
			}
			if got := slices.Contains(types.AdminRoles, tt.role); got != tt.isAdmin {
				t.Errorf("in AdminRoles = %v, want %v", got, tt.isAdmin)
			}
		})
	}
}
