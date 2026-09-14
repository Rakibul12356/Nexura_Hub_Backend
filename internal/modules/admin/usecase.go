// internal/modules/admin/usecase.go
package admin

import (
	"context"

	"github.com/google/uuid"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/course"
)

type AdminUsecase interface {
	GetOverviewStats(ctx context.Context) (*AdminOverviewStats, error)
	ApproveCourse(ctx context.Context, courseID uuid.UUID) error
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status auth.UserStatus) error
}

type adminUsecase struct {
	adminRepo  AdminRepository
	courseRepo course.CourseRepository
	userRepo   auth.UserRepository
}

func NewAdminUsecase(adminRepo AdminRepository, courseRepo course.CourseRepository, userRepo auth.UserRepository) AdminUsecase {
	return &adminUsecase{
		adminRepo:  adminRepo,
		courseRepo: courseRepo,
		userRepo:   userRepo,
	}
}

func (u *adminUsecase) GetOverviewStats(ctx context.Context) (*AdminOverviewStats, error) {
	return u.adminRepo.GetAdminOverviewStats(ctx)
}

func (u *adminUsecase) ApproveCourse(ctx context.Context, courseID uuid.UUID) error {
	return u.courseRepo.SetPublished(ctx, courseID, true)
}

func (u *adminUsecase) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status auth.UserStatus) error {
	return u.userRepo.UpdateStatus(ctx, userID, status)
}
