package instructor

type InstructorStats struct {
	TotalCourses       int     `json:"totalCourses"`
	TotalEnrollments   int     `json:"totalEnrollments"`
	TotalRevenue       float64 `json:"totalRevenue"`
	InstructorEarnings float64 `json:"instructorEarnings"`
	TotalStudents      int     `json:"totalStudents"`
}
