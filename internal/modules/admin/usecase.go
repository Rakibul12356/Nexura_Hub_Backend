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
	GetRevenueAnalytics(ctx context.Context) (map[string]interface{}, error)
	ListUsers(ctx context.Context, role string) ([]*auth.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*auth.User, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role auth.UserRole) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
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

func (u *adminUsecase) GetRevenueAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"totalGMV":         12450.00,
		"adminNetCut":      622.50,
		"instructorPayout": 11827.50,
		"transactions":     []interface{}{},
	}, nil
}

func (u *adminUsecase) ListUsers(ctx context.Context, role string) ([]*auth.User, error) {
	return []*auth.User{}, nil
}

func (u *adminUsecase) GetUserByID(ctx context.Context, userID uuid.UUID) (*auth.User, error) {
	return u.userRepo.GetByID(ctx, userID)
}

func (u *adminUsecase) UpdateUserRole(ctx context.Context, userID uuid.UUID, role auth.UserRole) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Role = role
	return u.userRepo.Update(ctx, user)
}

func (u *adminUsecase) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return u.userRepo.UpdateStatus(ctx, userID, auth.StatusSuspended)
}
