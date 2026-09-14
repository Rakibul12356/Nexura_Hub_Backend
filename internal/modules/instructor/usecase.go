// internal/modules/instructor/usecase.go
package instructor

import (
	"context"

	"github.com/google/uuid"
)

type InstructorUsecase interface {
	GetDashboardStats(ctx context.Context, instructorID uuid.UUID) (*InstructorStats, error)
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
