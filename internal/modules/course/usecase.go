package course

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/utils"
)

type CourseUsecase interface {
	ListCourses(ctx context.Context, f ListFilter) ([]Course, int, error)
	GetCourse(ctx context.Context, idOrSlug string, viewer Viewer) (*Course, error)
	CreateCourse(ctx context.Context, instructorID uuid.UUID, role string, dto CreateCourseDTO) (*Course, error)
	UpdateCourse(ctx context.Context, id uuid.UUID, viewer Viewer, dto UpdateCourseDTO) (*Course, error)
	DeleteCourse(ctx context.Context, id uuid.UUID, viewer Viewer) error
	TogglePublish(ctx context.Context, id uuid.UUID, viewer Viewer, published bool) error
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id uuid.UUID) (*Category, error)
	CreateCategory(ctx context.Context, dto Category) (*Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, dto Category) (*Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	GetModules(ctx context.Context, courseID uuid.UUID, viewer Viewer) ([]Module, error)
	GetModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer) (*Module, error)
	AddModule(ctx context.Context, courseID uuid.UUID, viewer Viewer, dto ModuleDTO) (*Module, error)
	UpdateModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer, dto ModuleDTO) (*Module, error)
	DeleteModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer) error
	ReorderModules(ctx context.Context, courseID uuid.UUID, viewer Viewer, ids []uuid.UUID) error
	GetLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer) (*Lesson, error)
	AddLesson(ctx context.Context, moduleID uuid.UUID, viewer Viewer, dto LessonDTO) (*Lesson, error)
	UpdateLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer, dto LessonDTO) (*Lesson, error)
	DeleteLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer) error
	ReorderLessons(ctx context.Context, moduleID uuid.UUID, viewer Viewer, ids []uuid.UUID) error
	AddResource(ctx context.Context, lessonID uuid.UUID, viewer Viewer, dto ResourceDTO) (*LessonResource, error)
	DeleteResource(ctx context.Context, lessonID, resourceID uuid.UUID, viewer Viewer) error
	GetReviews(ctx context.Context, courseID string) ([]CourseReview, error)
	AddReview(ctx context.Context, courseID string, userID uuid.UUID, dto ReviewDTO) (*CourseReview, error)
	UpdateReview(ctx context.Context, id uuid.UUID, viewer Viewer, dto ReviewDTO) (*CourseReview, error)
	DeleteReview(ctx context.Context, id uuid.UUID, viewer Viewer) error
	GetNotes(ctx context.Context, lessonID, userID uuid.UUID) ([]Note, error)
	AddNote(ctx context.Context, lessonID, userID uuid.UUID, dto NoteDTO) (*Note, error)
	DeleteNote(ctx context.Context, lessonID, noteID, userID uuid.UUID) error
	GetDiscussions(ctx context.Context, lessonID uuid.UUID, viewer Viewer) ([]Discussion, error)
	AddDiscussion(ctx context.Context, lessonID uuid.UUID, viewer Viewer, content string) (*Discussion, error)
	AddReply(ctx context.Context, discussionID uuid.UUID, viewer Viewer, content string) (*DiscussionReply, error)
	ToggleUpvote(ctx context.Context, discussionID uuid.UUID, userID uuid.UUID) (int, error)
	GetEnrolledCourses(ctx context.Context, userID uuid.UUID) ([]EnrolledCourse, error)
	CanManageCourse(ctx context.Context, courseID uuid.UUID, viewer Viewer) error
}

type courseUsecase struct {
	repo CourseRepository
}

func NewCourseUsecase(repo CourseRepository) CourseUsecase {
	return &courseUsecase{repo: repo}
}

func (u *courseUsecase) ListCourses(ctx context.Context, f ListFilter) ([]Course, int, error) {
	return u.repo.ListCourses(ctx, f)
}

func (u *courseUsecase) GetCourse(ctx context.Context, idOrSlug string, viewer Viewer) (*Course, error) {
	c, err := u.repo.GetByIDOrSlug(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	enrolled := false
	if viewer.UserID != uuid.Nil {
		enrolled, _ = u.repo.IsEnrolled(ctx, viewer.UserID, c.ID)
		c.Progress = u.repo.GetEnrollmentProgress(ctx, viewer.UserID, c.ID)
		done, _ := u.repo.CompletedLessonIDs(ctx, viewer.UserID, c.ID)
		for i := range c.Modules {
			for j := range c.Modules[i].Lessons {
				c.Modules[i].Lessons[j].Completed = done[c.Modules[i].Lessons[j].ID]
			}
		}
	}
	isOwner := viewer.UserID == c.InstructorID || viewer.Role == "admin"
	if !c.IsPublished && !isOwner {
		return nil, appErrors.ErrCourseNotFound
	}
	u.stripPaidVideos(c, enrolled, isOwner)
	return c, nil
}

func (u *courseUsecase) stripPaidVideos(c *Course, enrolled, isOwner bool) {
	for i := range c.Modules {
		for j := range c.Modules[i].Lessons {
			l := &c.Modules[i].Lessons[j]
			if !l.IsFree && !enrolled && !isOwner {
				l.VideoURL = nil
			}
		}
	}
}

func (u *courseUsecase) CreateCourse(ctx context.Context, instructorID uuid.UUID, role string, dto CreateCourseDTO) (*Course, error) {
	creator := "instructor"
	if role == "admin" {
		creator = "admin"
	}
	catID, catTitle, _ := u.repo.FindCategoryID(ctx, dto.CategoryID, dto.Category)
	thumb := dto.Thumbnail
	if thumb == "" {
		thumb = dto.Image
	}
	if thumb == "" {
		thumb = "https://images.unsplash.com/photo-1633356122544-f134324a6cee"
	}
	c := &Course{
		ID:             uuid.New(),
		Slug:           utils.Slugify(dto.Title),
		Title:          dto.Title,
		Subtitle:       strPtr(dto.Subtitle),
		Description:    strPtr(dto.Description),
		CategoryID:     catID,
		Category:       catTitle,
		InstructorID:   instructorID,
		CreatorType:    creator,
		Thumbnail:      thumb,
		Image:          thumb,
		Price:          dto.Price,
		DiscountPrice:  dto.DiscountPrice,
		LearningPoints: dto.LearningPoints,
	}
	if c.LearningPoints == nil {
		c.LearningPoints = []string{}
	}
	if err := u.repo.CreateCourse(ctx, c); err != nil {
		return nil, err
	}
	return u.repo.GetByID(ctx, c.ID)
}

func (u *courseUsecase) CanManageCourse(ctx context.Context, courseID uuid.UUID, viewer Viewer) error {
	if viewer.Role == "admin" {
		return nil
	}
	owner, err := u.repo.OwnerIDForCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if owner != viewer.UserID {
		return appErrors.ErrForbidden
	}
	return nil
}

func (u *courseUsecase) UpdateCourse(ctx context.Context, id uuid.UUID, viewer Viewer, dto UpdateCourseDTO) (*Course, error) {
	if err := u.CanManageCourse(ctx, id, viewer); err != nil {
		return nil, err
	}
	c, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dto.Title != nil {
		c.Title = *dto.Title
	}
	if dto.Description != nil {
		c.Description = dto.Description
	}
	if dto.Subtitle != nil {
		c.Subtitle = dto.Subtitle
	}
	if dto.Thumbnail != nil {
		c.Thumbnail = *dto.Thumbnail
	}
	if dto.Image != nil && *dto.Image != "" {
		c.Thumbnail = *dto.Image
	}
	c.Image = c.Thumbnail
	if dto.Price != nil {
		c.Price = *dto.Price
	}
	if dto.DiscountPrice != nil {
		c.DiscountPrice = dto.DiscountPrice
	}
	if dto.IsPublished != nil {
		c.IsPublished = *dto.IsPublished
	}
	if dto.IsFeatured != nil && viewer.Role == "admin" {
		c.IsFeatured = *dto.IsFeatured
	}
	if dto.LearningPoints != nil {
		c.LearningPoints = dto.LearningPoints
	}
	if dto.CategoryID != nil || (dto.Category != nil && *dto.Category != "") {
		slug := ""
		if dto.Category != nil {
			slug = *dto.Category
		}
		catID, title, _ := u.repo.FindCategoryID(ctx, dto.CategoryID, slug)
		c.CategoryID = catID
		c.Category = title
	}
	if err := u.repo.UpdateCourse(ctx, c); err != nil {
		return nil, err
	}
	return u.repo.GetByID(ctx, id)
}

func (u *courseUsecase) DeleteCourse(ctx context.Context, id uuid.UUID, viewer Viewer) error {
	if err := u.CanManageCourse(ctx, id, viewer); err != nil {
		return err
	}
	return u.repo.SoftDeleteCourse(ctx, id)
}

func (u *courseUsecase) TogglePublish(ctx context.Context, id uuid.UUID, viewer Viewer, published bool) error {
	if err := u.CanManageCourse(ctx, id, viewer); err != nil {
		return err
	}
	return u.repo.SetPublished(ctx, id, published)
}

func (u *courseUsecase) GetCategories(ctx context.Context) ([]Category, error) {
	return u.repo.GetCategories(ctx)
}
func (u *courseUsecase) GetCategory(ctx context.Context, id uuid.UUID) (*Category, error) {
	return u.repo.GetCategory(ctx, id)
}
func (u *courseUsecase) CreateCategory(ctx context.Context, dto Category) (*Category, error) {
	if dto.Slug == "" {
		dto.Slug = strings.ToLower(strings.ReplaceAll(dto.Title, " ", "-"))
	}
	dto.Value, dto.Label = dto.Slug, dto.Title
	if err := u.repo.CreateCategory(ctx, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}
func (u *courseUsecase) UpdateCategory(ctx context.Context, id uuid.UUID, dto Category) (*Category, error) {
	dto.ID = id
	if dto.Slug == "" {
		dto.Slug = strings.ToLower(strings.ReplaceAll(dto.Title, " ", "-"))
	}
	dto.Value, dto.Label = dto.Slug, dto.Title
	if err := u.repo.UpdateCategory(ctx, &dto); err != nil {
		return nil, err
	}
	return u.repo.GetCategory(ctx, id)
}
func (u *courseUsecase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return u.repo.DeleteCategory(ctx, id)
}

func (u *courseUsecase) GetModules(ctx context.Context, courseID uuid.UUID, viewer Viewer) ([]Module, error) {
	return u.repo.GetModules(ctx, courseID)
}
func (u *courseUsecase) GetModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer) (*Module, error) {
	return u.repo.GetModule(ctx, moduleID)
}
func (u *courseUsecase) AddModule(ctx context.Context, courseID uuid.UUID, viewer Viewer, dto ModuleDTO) (*Module, error) {
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return nil, err
	}
	m := &Module{CourseID: courseID, Title: dto.Title, Description: strPtr(dto.Description), Lessons: []Lesson{}}
	if dto.IsPublished != nil {
		m.IsPublished = *dto.IsPublished
	}
	if err := u.repo.AddModule(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}
func (u *courseUsecase) UpdateModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer, dto ModuleDTO) (*Module, error) {
	courseID, err := u.repo.CourseIDForModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return nil, err
	}
	m, err := u.repo.GetModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if dto.Title != "" {
		m.Title = dto.Title
	}
	if dto.Description != "" {
		m.Description = strPtr(dto.Description)
	}
	if dto.IsPublished != nil {
		m.IsPublished = *dto.IsPublished
	}
	if err := u.repo.UpdateModule(ctx, m); err != nil {
		return nil, err
	}
	return u.repo.GetModule(ctx, moduleID)
}
func (u *courseUsecase) DeleteModule(ctx context.Context, moduleID uuid.UUID, viewer Viewer) error {
	courseID, err := u.repo.CourseIDForModule(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return err
	}
	return u.repo.DeleteModule(ctx, moduleID)
}
func (u *courseUsecase) ReorderModules(ctx context.Context, courseID uuid.UUID, viewer Viewer, ids []uuid.UUID) error {
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return err
	}
	return u.repo.ReorderModules(ctx, courseID, ids)
}

func (u *courseUsecase) GetLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer) (*Lesson, error) {
	l, err := u.repo.GetLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	owner, _ := u.repo.OwnerIDForCourse(ctx, courseID)
	enrolled, _ := u.repo.IsEnrolled(ctx, viewer.UserID, courseID)
	isOwner := viewer.Role == "admin" || viewer.UserID == owner
	if !l.IsFree && !enrolled && !isOwner {
		l.VideoURL = nil
	}
	if enrolled {
		done, _ := u.repo.CompletedLessonIDs(ctx, viewer.UserID, courseID)
		l.Completed = done[l.ID]
	}
	return l, nil
}

func (u *courseUsecase) AddLesson(ctx context.Context, moduleID uuid.UUID, viewer Viewer, dto LessonDTO) (*Lesson, error) {
	courseID, err := u.repo.CourseIDForModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return nil, err
	}
	l := &Lesson{
		ModuleID: moduleID, Title: dto.Title, Description: strPtr(dto.Description),
		VideoURL: strPtr(dto.VideoURL), Duration: strPtr(dto.Duration), QuizSetID: dto.QuizSetID, Resources: []LessonResource{},
	}
	if l.Title == "" {
		return nil, errors.New("title is required")
	}
	if dto.IsFree != nil {
		l.IsFree = *dto.IsFree
	}
	if dto.IsPublished != nil {
		l.IsPublished = *dto.IsPublished
	}
	if err := u.repo.AddLesson(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (u *courseUsecase) UpdateLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer, dto LessonDTO) (*Lesson, error) {
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return nil, err
	}
	l, err := u.repo.GetLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if dto.Title != "" {
		l.Title = dto.Title
	}
	if dto.Description != "" {
		l.Description = strPtr(dto.Description)
	}
	if dto.VideoURL != "" {
		l.VideoURL = strPtr(dto.VideoURL)
	}
	if dto.Duration != "" {
		l.Duration = strPtr(dto.Duration)
	}
	if dto.IsFree != nil {
		l.IsFree = *dto.IsFree
	}
	if dto.IsPublished != nil {
		l.IsPublished = *dto.IsPublished
	}
	if dto.QuizSetID != nil {
		l.QuizSetID = dto.QuizSetID
	}
	if err := u.repo.UpdateLesson(ctx, l); err != nil {
		return nil, err
	}
	return u.repo.GetLesson(ctx, lessonID)
}

func (u *courseUsecase) DeleteLesson(ctx context.Context, lessonID uuid.UUID, viewer Viewer) error {
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return err
	}
	return u.repo.DeleteLesson(ctx, lessonID)
}

func (u *courseUsecase) ReorderLessons(ctx context.Context, moduleID uuid.UUID, viewer Viewer, ids []uuid.UUID) error {
	courseID, err := u.repo.CourseIDForModule(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return err
	}
	return u.repo.ReorderLessons(ctx, moduleID, ids)
}

func (u *courseUsecase) AddResource(ctx context.Context, lessonID uuid.UUID, viewer Viewer, dto ResourceDTO) (*LessonResource, error) {
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return nil, err
	}
	res := &LessonResource{Title: dto.Title, Type: dto.Type, URL: strPtr(dto.URL), Size: strPtr(dto.Size), FileName: strPtr(dto.FileName), Content: strPtr(dto.Content)}
	if err := u.repo.AddResource(ctx, lessonID, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (u *courseUsecase) DeleteResource(ctx context.Context, lessonID, resourceID uuid.UUID, viewer Viewer) error {
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := u.CanManageCourse(ctx, courseID, viewer); err != nil {
		return err
	}
	return u.repo.DeleteResource(ctx, lessonID, resourceID)
}

func (u *courseUsecase) GetReviews(ctx context.Context, courseID string) ([]CourseReview, error) {
	c, err := u.repo.GetByIDOrSlug(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return u.repo.GetReviews(ctx, c.ID)
}

func (u *courseUsecase) AddReview(ctx context.Context, courseID string, userID uuid.UUID, dto ReviewDTO) (*CourseReview, error) {
	c, err := u.repo.GetByIDOrSlug(ctx, courseID)
	if err != nil {
		return nil, err
	}
	ok, _ := u.repo.IsEnrolled(ctx, userID, c.ID)
	if !ok {
		return nil, appErrors.ErrNotEnrolled
	}
	return u.repo.UpsertReview(ctx, c.ID, userID, dto.Rating, dto.Comment)
}

func (u *courseUsecase) UpdateReview(ctx context.Context, id uuid.UUID, viewer Viewer, dto ReviewDTO) (*CourseReview, error) {
	return u.repo.UpdateReview(ctx, id, viewer.UserID, dto.Rating, dto.Comment, viewer.Role == "admin")
}
func (u *courseUsecase) DeleteReview(ctx context.Context, id uuid.UUID, viewer Viewer) error {
	return u.repo.DeleteReview(ctx, id, viewer.UserID, viewer.Role == "admin")
}

func (u *courseUsecase) GetNotes(ctx context.Context, lessonID, userID uuid.UUID) ([]Note, error) {
	return u.repo.GetNotes(ctx, lessonID, userID)
}
func (u *courseUsecase) AddNote(ctx context.Context, lessonID, userID uuid.UUID, dto NoteDTO) (*Note, error) {
	return u.repo.AddNote(ctx, lessonID, userID, dto.Timestamp, dto.Text)
}
func (u *courseUsecase) DeleteNote(ctx context.Context, lessonID, noteID, userID uuid.UUID) error {
	return u.repo.DeleteNote(ctx, lessonID, noteID, userID)
}

func (u *courseUsecase) GetDiscussions(ctx context.Context, lessonID uuid.UUID, viewer Viewer) ([]Discussion, error) {
	return u.repo.GetDiscussions(ctx, lessonID)
}
func (u *courseUsecase) AddDiscussion(ctx context.Context, lessonID uuid.UUID, viewer Viewer, content string) (*Discussion, error) {
	courseID, _, err := u.repo.CourseIDForLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	return u.repo.AddDiscussion(ctx, lessonID, courseID, viewer.UserID, content)
}
func (u *courseUsecase) AddReply(ctx context.Context, discussionID uuid.UUID, viewer Viewer, content string) (*DiscussionReply, error) {
	return u.repo.AddReply(ctx, discussionID, viewer.UserID, content)
}
func (u *courseUsecase) ToggleUpvote(ctx context.Context, discussionID uuid.UUID, userID uuid.UUID) (int, error) {
	return u.repo.ToggleUpvote(ctx, discussionID, userID)
}
func (u *courseUsecase) GetEnrolledCourses(ctx context.Context, userID uuid.UUID) ([]EnrolledCourse, error) {
	return u.repo.GetEnrolledCourses(ctx, userID)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
