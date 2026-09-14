// internal/modules/instructor/repository.go
package instructor

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type InstructorRepository interface {
	GetInstructorStats(ctx context.Context, instructorID uuid.UUID) (*InstructorStats, error)
}

type postgresInstructorRepository struct {
	db *sql.DB
}

func NewInstructorRepository(db *sql.DB) InstructorRepository {
	return &postgresInstructorRepository{db: db}
}

func (r *postgresInstructorRepository) GetInstructorStats(ctx context.Context, instructorID uuid.UUID) (*InstructorStats, error) {
	earningsQuery := `
		SELECT COALESCE(SUM(instructor_earnings), 0.00)
		FROM transactions
		WHERE instructor_id = $1 AND status = 'completed'
	`
	var totalEarnings float64
	_ = r.db.QueryRowContext(ctx, earningsQuery, instructorID).Scan(&totalEarnings)

	studentsQuery := `
		SELECT COUNT(DISTINCT student_id)
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE c.instructor_id = $1 AND c.deleted_at IS NULL
	`
	var activeStudents int
	_ = r.db.QueryRowContext(ctx, studentsQuery, instructorID).Scan(&activeStudents)

	coursesQuery := `SELECT COUNT(*) FROM courses WHERE instructor_id = $1 AND deleted_at IS NULL`
	var totalCourses int
	_ = r.db.QueryRowContext(ctx, coursesQuery, instructorID).Scan(&totalCourses)

	recentQuery := `
		SELECT u.first_name || ' ' || u.last_name AS student_name, c.title, t.gross_amount, t.instructor_earnings, t.created_at
		FROM transactions t
		JOIN users u ON t.student_id = u.id
		JOIN courses c ON t.course_id = c.id
		WHERE t.instructor_id = $1
		ORDER BY t.created_at DESC
		LIMIT 5
	`
	rows, err := r.db.QueryContext(ctx, recentQuery, instructorID)
	var recent []RecentEnrollment
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rec RecentEnrollment
			if err := rows.Scan(&rec.StudentName, &rec.CourseTitle, &rec.Price, &rec.InstructorNet, &rec.Date); err == nil {
				recent = append(recent, rec)
			}
		}
	}
	if recent == nil {
		recent = []RecentEnrollment{}
	}

	return &InstructorStats{
		TotalEarnings:     totalEarnings,
		ActiveStudents:    activeStudents,
		TotalCourses:      totalCourses,
		RecentEnrollments: recent,
	}, nil
}
