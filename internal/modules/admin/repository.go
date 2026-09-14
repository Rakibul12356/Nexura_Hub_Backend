// internal/modules/admin/repository.go
package admin

import (
	"context"
	"database/sql"
)

type AdminRepository interface {
	GetAdminOverviewStats(ctx context.Context) (*AdminOverviewStats, error)
}

type postgresAdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) AdminRepository {
	return &postgresAdminRepository{db: db}
}

func (r *postgresAdminRepository) GetAdminOverviewStats(ctx context.Context) (*AdminOverviewStats, error) {
	var totalRevenue float64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(gross_amount), 0.00) FROM transactions WHERE status = 'completed'`).Scan(&totalRevenue)

	var adminNetCommission float64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(admin_commission_amount), 0.00) FROM transactions WHERE status = 'completed'`).Scan(&adminNetCommission)

	var instructorPayouts float64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(instructor_earnings), 0.00) FROM transactions WHERE status = 'completed'`).Scan(&instructorPayouts)

	var totalStudents, totalInstructors, totalCourses int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'student' AND deleted_at IS NULL`).Scan(&totalStudents)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'instructor' AND deleted_at IS NULL`).Scan(&totalInstructors)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`).Scan(&totalCourses)

	return &AdminOverviewStats{
		TotalRevenue:       totalRevenue,
		AdminNetCommission: adminNetCommission,
		InstructorPayouts:  instructorPayouts,
		TotalStudents:      totalStudents,
		TotalInstructors:   totalInstructors,
		TotalCourses:       totalCourses,
	}, nil
}
