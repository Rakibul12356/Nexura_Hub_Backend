// internal/modules/instructor/models.go
package instructor

import (
	"time"
)

type RecentEnrollment struct {
	StudentName   string    `json:"studentName"`
	CourseTitle   string    `json:"courseTitle"`
	Price         float64   `json:"price"`
	InstructorNet float64   `json:"instructorNet"`
	Date          time.Time `json:"date"`
}

type InstructorStats struct {
	TotalEarnings     float64            `json:"totalEarnings"`
	ActiveStudents    int                `json:"activeStudents"`
	TotalCourses      int                `json:"totalCourses"`
	RecentEnrollments []RecentEnrollment `json:"recentEnrollments"`
}
