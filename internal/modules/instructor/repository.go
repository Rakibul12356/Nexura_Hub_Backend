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
	s := &InstructorStats{}
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE instructor_id=$1 AND deleted_at IS NULL`, instructorID).Scan(&s.TotalCourses)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments e JOIN courses c ON c.id=e.course_id WHERE c.instructor_id=$1`, instructorID).Scan(&s.TotalEnrollments)
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE instructor_id=$1 AND status='completed'`, instructorID).Scan(&s.TotalRevenue, &s.InstructorEarnings)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses c ON c.id=e.course_id WHERE c.instructor_id=$1`, instructorID).Scan(&s.TotalStudents)
	return s, nil
}
