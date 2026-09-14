// internal/modules/course/usecase.go
package course

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type CourseUsecase interface {
	ListPublicCourses(ctx context.Context, search, category string, page, limit int) ([]Course, int, error)
	GetCourseBySlug(ctx context.Context, slug string) (*Course, error)
	GetCourseByID(ctx context.Context, id uuid.UUID) (*Course, error)
	GetCategories(ctx context.Context) ([]Category, error)
	CreateCourse(ctx context.Context, instructorID uuid.UUID, dto CreateCourseDTO) (*Course, error)
}

type courseUsecase struct {
	courseRepo CourseRepository
}

func NewCourseUsecase(courseRepo CourseRepository) CourseUsecase {
	return &courseUsecase{courseRepo: courseRepo}
}

func (u *courseUsecase) ListPublicCourses(ctx context.Context, search, category string, page, limit int) ([]Course, int, error) {
	return u.courseRepo.ListCourses(ctx, search, category, page, limit)
}

func (u *courseUsecase) GetCourseBySlug(ctx context.Context, slug string) (*Course, error) {
	return u.courseRepo.GetBySlug(ctx, slug)
}

func (u *courseUsecase) GetCourseByID(ctx context.Context, id uuid.UUID) (*Course, error) {
	return u.courseRepo.GetByID(ctx, id)
}

func (u *courseUsecase) GetCategories(ctx context.Context) ([]Category, error) {
	return u.courseRepo.GetCategories(ctx)
}

func (u *courseUsecase) CreateCourse(ctx context.Context, instructorID uuid.UUID, dto CreateCourseDTO) (*Course, error) {
	slug := strings.ToLower(strings.ReplaceAll(dto.Title, " ", "-"))
	slug = fmt.Sprintf("%s-%d", slug, uuid.New().ID()%10000)

	course := &Course{
		ID:             uuid.New(),
		Title:          dto.Title,
		Slug:           slug,
		Subtitle:       &dto.Subtitle,
		Description:    &dto.Description,
		CategoryID:     dto.CategoryID,
		InstructorID:   instructorID,
		Thumbnail:      dto.Thumbnail,
		Price:          dto.Price,
		DiscountPrice:  dto.DiscountPrice,
		IsPublished:    true,
		LearningPoints: dto.LearningPoints,
	}

	if course.Thumbnail == "" {
		course.Thumbnail = "https://images.unsplash.com/photo-1633356122544-f134324a6cee"
	}

	if err := u.courseRepo.CreateCourse(ctx, course); err != nil {
		return nil, err
	}

	return course, nil
}
