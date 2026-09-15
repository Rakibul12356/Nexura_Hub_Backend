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
	UpdateCourse(ctx context.Context, id uuid.UUID, dto CreateCourseDTO) (*Course, error)
	DeleteCourse(ctx context.Context, id uuid.UUID) error
	TogglePublish(ctx context.Context, id uuid.UUID, isPublished bool) error
	AddModule(ctx context.Context, courseID uuid.UUID, title string) (map[string]interface{}, error)
	UpdateModule(ctx context.Context, moduleID uuid.UUID, title string) error
	DeleteModule(ctx context.Context, moduleID uuid.UUID) error
	AddLesson(ctx context.Context, moduleID uuid.UUID, title, videoURL string) (map[string]interface{}, error)
	UpdateLesson(ctx context.Context, lessonID uuid.UUID, title, videoURL string) error
	DeleteLesson(ctx context.Context, lessonID uuid.UUID) error
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

func (u *courseUsecase) UpdateCourse(ctx context.Context, id uuid.UUID, dto CreateCourseDTO) (*Course, error) {
	course, err := u.courseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	course.Title = dto.Title
	course.Subtitle = &dto.Subtitle
	course.Description = &dto.Description
	course.Price = dto.Price
	course.DiscountPrice = dto.DiscountPrice
	if dto.Thumbnail != "" {
		course.Thumbnail = dto.Thumbnail
	}
	return course, nil
}

func (u *courseUsecase) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (u *courseUsecase) TogglePublish(ctx context.Context, id uuid.UUID, isPublished bool) error {
	return u.courseRepo.SetPublished(ctx, id, isPublished)
}

func (u *courseUsecase) AddModule(ctx context.Context, courseID uuid.UUID, title string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":       uuid.New().String(),
		"courseId": courseID,
		"title":    title,
		"lessons":  []interface{}{},
	}, nil
}

func (u *courseUsecase) UpdateModule(ctx context.Context, moduleID uuid.UUID, title string) error {
	return nil
}

func (u *courseUsecase) DeleteModule(ctx context.Context, moduleID uuid.UUID) error {
	return nil
}

func (u *courseUsecase) AddLesson(ctx context.Context, moduleID uuid.UUID, title, videoURL string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":       uuid.New().String(),
		"moduleId": moduleID,
		"title":    title,
		"videoUrl": videoURL,
	}, nil
}

func (u *courseUsecase) UpdateLesson(ctx context.Context, lessonID uuid.UUID, title, videoURL string) error {
	return nil
}

func (u *courseUsecase) DeleteLesson(ctx context.Context, lessonID uuid.UUID) error {
	return nil
}
