// internal/modules/course/models.go
package course

import (
	"time"

	"github.com/google/uuid"
	"nexura-backend/internal/modules/auth"
)

type Category struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Thumbnail *string   `json:"thumbnail"`
	CreatedAt time.Time `json:"createdAt"`
}

type CourseInstructorInfo struct {
	ID     uuid.UUID     `json:"id"`
	Name   string        `json:"name"`
	Avatar *string       `json:"avatar"`
	Role   auth.UserRole `json:"role"`
}

type LessonResource struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
	Size  string `json:"size"`
}

type Lesson struct {
	ID          uuid.UUID        `json:"id"`
	ModuleID    uuid.UUID        `json:"moduleId"`
	Title       string           `json:"title"`
	Description *string          `json:"description"`
	VideoURL    *string          `json:"videoUrl"`
	Duration    *string          `json:"duration"`
	IsFree      bool             `json:"isFree"`
	IsPublished bool             `json:"isPublished"`
	Position    int              `json:"position"`
	Resources   []LessonResource `json:"resources"`
	Completed   bool             `json:"completed,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

type Module struct {
	ID          uuid.UUID `json:"id"`
	CourseID    uuid.UUID `json:"courseId"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Position    int       `json:"position"`
	IsPublished bool      `json:"isPublished"`
	Lessons     []Lesson  `json:"lessons"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Course struct {
	ID             uuid.UUID             `json:"id"`
	Title          string                `json:"title"`
	Slug           string                `json:"slug"`
	Subtitle       *string               `json:"subtitle"`
	Description    *string               `json:"description"`
	CategoryID     *int                  `json:"categoryId"`
	CategoryName   string                `json:"category,omitempty"`
	InstructorID   uuid.UUID             `json:"instructorId"`
	Instructor     *CourseInstructorInfo `json:"instructor,omitempty"`
	Thumbnail      string                `json:"thumbnail"`
	Price          float64               `json:"price"`
	DiscountPrice  *float64              `json:"discountPrice"`
	IsPublished    bool                  `json:"isPublished"`
	LearningPoints []string              `json:"learningPoints"`
	Modules        []Module              `json:"modules,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
	DeletedAt      *time.Time            `json:"-"`
}

type CreateCourseDTO struct {
	Title          string   `json:"title" binding:"required"`
	Category       string   `json:"category"`
	CategoryID     *int     `json:"categoryId"`
	Price          float64  `json:"price"`
	DiscountPrice  *float64 `json:"discountPrice"`
	Description    string   `json:"description"`
	Subtitle       string   `json:"subtitle"`
	Thumbnail      string   `json:"thumbnail"`
	LearningPoints []string `json:"learningPoints"`
}
