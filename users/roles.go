// Package users holds the user types shared with core.
package users

// Role is a user's role. The values are stored in the database and in the
// Kratos identity traits, so they never change.
type Role string

// User roles.
const (
	SystemAdmin     Role = "system_admin"
	FacilityAdmin   Role = "facility_admin"
	DepartmentAdmin Role = "department_admin"
	Student         Role = "student"
)

// AdminRoles lists every role except Student.
var AdminRoles = []Role{SystemAdmin, FacilityAdmin, DepartmentAdmin}
