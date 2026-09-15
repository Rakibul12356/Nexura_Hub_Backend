// internal/modules/instructor/usecase.go
package instructor

import (
	"context"

	"github.com/google/uuid"
)

type InstructorUsecase interface {
	GetDashboardStats(ctx context.Context, instructorID uuid.UUID) (*InstructorStats, error)
	GetInstructorCourses(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error)
	GetInstructorLives(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error)
	GetInstructorQuizSets(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error)
	GetInstructorEnrollments(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error)
}

type instructorUsecase struct {
	instructorRepo InstructorRepository
}

func NewInstructorUsecase(instructorRepo InstructorRepository) InstructorUsecase {
	return &instructorUsecase{instructorRepo: instructorRepo}
}

func (u *instructorUsecase) GetDashboardStats(ctx context.Context, instructorID uuid.UUID) (*InstructorStats, error) {
	return u.instructorRepo.GetInstructorStats(ctx, instructorID)
}

func (u *instructorUsecase) GetInstructorCourses(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (u *instructorUsecase) GetInstructorLives(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (u *instructorUsecase) GetInstructorQuizSets(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (u *instructorUsecase) GetInstructorEnrollments(ctx context.Context, instructorID uuid.UUID) ([]interface{}, error) {
	return []interface{}{}, nil
}
