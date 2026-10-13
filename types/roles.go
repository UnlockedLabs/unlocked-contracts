// Package types holds the basic user and facility types shared with core.
package types

// UserRole is a user's role. The values are stored in the database and in the
// Kratos identity traits, so they never change.
type UserRole string

// User roles.
const (
	SystemAdmin     UserRole = "system_admin"
	FacilityAdmin   UserRole = "facility_admin"
	DepartmentAdmin UserRole = "department_admin"
	Student         UserRole = "student"
)

// AdminRoles lists every role except Student.
var AdminRoles = []UserRole{SystemAdmin, FacilityAdmin, DepartmentAdmin}
