// internal/modules/enrollment/usecase.go
package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type EnrollmentUsecase interface {
	EnrollStudent(ctx context.Context, studentID, courseID uuid.UUID, paymentMethod string) (*EnrollResponseDTO, error)
	GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)
	CompleteLesson(ctx context.Context, studentID, lessonID uuid.UUID) error
}

type enrollmentUsecase struct {
	enrollmentRepo EnrollmentRepository
}

func NewEnrollmentUsecase(enrollmentRepo EnrollmentRepository) EnrollmentUsecase {
	return &enrollmentUsecase{
		enrollmentRepo: enrollmentRepo,
	}
}

func (u *enrollmentUsecase) EnrollStudent(ctx context.Context, studentID, courseID uuid.UUID, paymentMethod string) (*EnrollResponseDTO, error) {
	return u.enrollmentRepo.ExecuteEnrollmentTx(ctx, studentID, courseID, paymentMethod)
}

func (u *enrollmentUsecase) GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error) {
	return u.enrollmentRepo.GetEnrolledCourses(ctx, studentID)
}

func (u *enrollmentUsecase) CompleteLesson(ctx context.Context, studentID, lessonID uuid.UUID) error {
	return u.enrollmentRepo.UpdateLessonProgress(ctx, studentID, lessonID, true)
}
