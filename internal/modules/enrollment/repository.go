// internal/modules/enrollment/repository.go
package enrollment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/modules/course"
)

type EnrollmentRepository interface {
	IsEnrolled(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	GetEnrolledCourses(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)
	UpdateLessonProgress(ctx context.Context, studentID, lessonID uuid.UUID, completed bool) error
	ExecuteEnrollmentTx(ctx context.Context, studentID, courseID uuid.UUID, paymentMethod string) (*EnrollResponseDTO, error)
}

type postgresEnrollmentRepository struct {
	db *sql.DB
}

func NewEnrollmentRepository(db *sql.DB) EnrollmentRepository {
	return &postgresEnrollmentRepository{db: db}
}

func (r *postgresEnrollmentRepository) IsEnrolled(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id = $1 AND course_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, studentID, courseID).Scan(&exists)
	return exists, err
}

func (r *postgresEnrollmentRepository) GetEnrolledCourses(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error) {
	query := `
		SELECT e.id, e.student_id, e.course_id, e.enrolled_at, e.completed_at, e.progress_percentage,
		       c.title, c.slug, c.subtitle, c.thumbnail, c.price
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE e.student_id = $1 AND c.deleted_at IS NULL
		ORDER BY e.enrolled_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []Enrollment
	for rows.Next() {
		var enr Enrollment
		var c course.Course
		err := rows.Scan(
			&enr.ID, &enr.StudentID, &enr.CourseID, &enr.EnrolledAt, &enr.CompletedAt, &enr.ProgressPercentage,
			&c.Title, &c.Slug, &c.Subtitle, &c.Thumbnail, &c.Price,
		)
		if err != nil {
			return nil, err
		}
		c.ID = enr.CourseID
		enr.Course = &c
		enrollments = append(enrollments, enr)
	}

	return enrollments, nil
}

func (r *postgresEnrollmentRepository) UpdateLessonProgress(ctx context.Context, studentID, lessonID uuid.UUID, completed bool) error {
	query := `
		INSERT INTO lesson_progress (id, student_id, lesson_id, completed, completed_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (student_id, lesson_id)
		DO UPDATE SET completed = EXCLUDED.completed, completed_at = CURRENT_TIMESTAMP
	`
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, query, id, studentID, lessonID, completed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

func (r *postgresEnrollmentRepository) ExecuteEnrollmentTx(ctx context.Context, studentID, courseID uuid.UUID, paymentMethod string) (*EnrollResponseDTO, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var coursePrice float64
	var instructorID uuid.UUID
	var instructorRole string
	err = tx.QueryRowContext(ctx, `
		SELECT c.price, c.instructor_id, u.role 
		FROM courses c 
		JOIN users u ON c.instructor_id = u.id 
		WHERE c.id = $1 AND c.is_published = true AND c.deleted_at IS NULL
	`, courseID).Scan(&coursePrice, &instructorID, &instructorRole)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrCourseNotFound
		}
		return nil, err
	}

	var alreadyEnrolled bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id = $1 AND course_id = $2)`, studentID, courseID).Scan(&alreadyEnrolled)
	if err != nil {
		return nil, err
	}
	if alreadyEnrolled {
		return nil, appErrors.ErrAlreadyEnrolled
	}

	var commissionRate float64 = 0.05
	var creatorType CreatorType = CreatorInstructor
	if instructorRole == "admin" {
		commissionRate = 1.00
		creatorType = CreatorAdmin
	}

	adminAmount := coursePrice * commissionRate
	instructorAmount := coursePrice - adminAmount

	txID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (id, course_id, student_id, instructor_id, creator_type, gross_amount, admin_commission_rate, admin_commission_amount, instructor_earnings, status, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'completed', $10)
	`, txID, courseID, studentID, instructorID, creatorType, coursePrice, commissionRate, adminAmount, instructorAmount, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transaction: %w", err)
	}

	enrollmentID := uuid.New()
	_, err = tx.ExecContext(ctx, `INSERT INTO enrollments (id, student_id, course_id) VALUES ($1, $2, $3)`, enrollmentID, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert enrollment: %w", err)
	}

	var groupConvID *uuid.UUID
	var convID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT id FROM conversations WHERE course_id = $1 LIMIT 1`, courseID).Scan(&convID)
	if err == nil {
		groupConvID = &convID
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO conversation_members (conversation_id, user_id) 
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, convID, studentID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &EnrollResponseDTO{
		EnrollmentID:        enrollmentID,
		CourseID:            courseID,
		GroupConversationID: groupConvID,
	}, nil
}
