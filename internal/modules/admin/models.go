// internal/modules/admin/models.go
package admin

type AdminOverviewStats struct {
	TotalRevenue       float64 `json:"totalRevenue"`
	AdminNetCommission float64 `json:"adminNetCommission"`
	InstructorPayouts  float64 `json:"instructorPayouts"`
	TotalStudents      int     `json:"totalStudents"`
	TotalInstructors   int     `json:"totalInstructors"`
	TotalCourses       int     `json:"totalCourses"`
}
