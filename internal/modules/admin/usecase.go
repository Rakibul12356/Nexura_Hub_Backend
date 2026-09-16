package admin

import (
	"context"

	"github.com/google/uuid"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/course"
)

type AdminUsecase interface {
	ApproveCourse(ctx context.Context, courseID uuid.UUID) error
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status auth.UserStatus) error
}

type adminUsecase struct {
	courseRepo course.CourseRepository
	userRepo   auth.UserRepository
}

func NewAdminUsecase(courseRepo course.CourseRepository, userRepo auth.UserRepository) AdminUsecase {
	return &adminUsecase{courseRepo: courseRepo, userRepo: userRepo}
}

func (u *adminUsecase) ApproveCourse(ctx context.Context, courseID uuid.UUID) error {
	return u.courseRepo.SetPublished(ctx, courseID, true)
}

func (u *adminUsecase) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status auth.UserStatus) error {
	return u.userRepo.UpdateStatus(ctx, userID, status)
}
