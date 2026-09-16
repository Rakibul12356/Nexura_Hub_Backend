package course

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Thumbnail string    `json:"thumbnail"`
	Value     string    `json:"value"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

type CourseInstructorInfo struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Designation   string    `json:"designation,omitempty"`
	Avatar        *string   `json:"avatar"`
	Bio           *string   `json:"bio,omitempty"`
	Rating        float64   `json:"rating"`
	StudentsCount int       `json:"studentsCount"`
	CoursesCount  int       `json:"coursesCount"`
	ReviewsCount  int       `json:"reviewsCount"`
	Role          string    `json:"role,omitempty"`
}

type LessonResource struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Type     string    `json:"type"`
	URL      *string   `json:"url"`
	Size     *string   `json:"size"`
	FileName *string   `json:"fileName,omitempty"`
	Content  *string   `json:"content,omitempty"`
}

type QuizQuestionPublic struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description *string      `json:"description"`
	Points      int          `json:"points"`
	Options     []QuizOption `json:"options"`
}

type QuizOption struct {
	ID        uuid.UUID `json:"id"`
	Label     string    `json:"label"`
	IsCorrect *bool     `json:"isCorrect,omitempty"`
	Position  int       `json:"position,omitempty"`
}

type Lesson struct {
	ID           uuid.UUID            `json:"id"`
	ModuleID     uuid.UUID            `json:"moduleId"`
	Title        string               `json:"title"`
	Description  *string              `json:"description"`
	VideoURL     *string              `json:"videoUrl"`
	Duration     *string              `json:"duration"`
	IsFree       bool                 `json:"isFree"`
	IsPublished  bool                 `json:"isPublished"`
	Position     int                  `json:"position"`
	Completed    bool                 `json:"completed"`
	QuizSetID    *uuid.UUID           `json:"quizSetId"`
	QuizSetTitle *string              `json:"quizSetTitle,omitempty"`
	HasQuiz      bool                 `json:"hasQuiz"`
	Questions    []QuizQuestionPublic `json:"questions,omitempty"`
	Resources    []LessonResource     `json:"resources"`
	CreatedAt    time.Time            `json:"createdAt,omitempty"`
}

type Module struct {
	ID          uuid.UUID `json:"id"`
	CourseID    uuid.UUID `json:"courseId"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Position    int       `json:"position"`
	IsPublished bool      `json:"isPublished"`
	Lessons     []Lesson  `json:"lessons"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

type CourseReview struct {
	ID            uuid.UUID `json:"id"`
	StudentName   string    `json:"studentName"`
	StudentAvatar *string   `json:"studentAvatar"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	Date          string    `json:"date"`
	UserID        uuid.UUID `json:"userId,omitempty"`
}

type Course struct {
	ID              uuid.UUID             `json:"id"`
	Slug            string                `json:"slug"`
	Title           string                `json:"title"`
	Subtitle        *string               `json:"subtitle"`
	Description     *string               `json:"description"`
	Category        string                `json:"category"`
	CategoryID      *uuid.UUID            `json:"categoryId"`
	Thumbnail       string                `json:"thumbnail"`
	Image           string                `json:"image"`
	Price           float64               `json:"price"`
	DiscountPrice   *float64              `json:"discountPrice"`
	IsPublished     bool                  `json:"isPublished"`
	IsFeatured      bool                  `json:"isFeatured"`
	TotalChapters   int                   `json:"totalChapters"`
	Progress        float64               `json:"progress"`
	CreatorType     string                `json:"creatorType"`
	LearningPoints  []string              `json:"learningPoints"`
	InstructorID    uuid.UUID             `json:"instructorId"`
	Instructor      *CourseInstructorInfo `json:"instructor,omitempty"`
	Modules         []Module              `json:"modules,omitempty"`
	QuizSets        []any                 `json:"quizSets,omitempty"`
	Reviews         []CourseReview        `json:"reviews,omitempty"`
	EnrollmentsCount int                  `json:"enrollmentsCount,omitempty"`
	TotalRevenue    float64               `json:"totalRevenue,omitempty"`
	AdminEarnings   float64               `json:"adminEarnings,omitempty"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type EnrolledCourse struct {
	Course
	EnrolledDate     string  `json:"enrolledDate"`
	CompletedModules int     `json:"completedModules"`
	TotalModules     int     `json:"totalModules"`
	CompletedQuizzes int     `json:"completedQuizzes"`
	TotalQuizzes     int     `json:"totalQuizzes"`
	QuizScore        float64 `json:"quizScore"`
	OtherScore       float64 `json:"otherScore"`
	TotalScore       float64 `json:"totalScore"`
	PaymentStatus    string  `json:"paymentStatus"`
}

type ListFilter struct {
	Search             string
	Category           string
	Price              string
	Sort               string
	IsFeatured         *bool
	IsPublished        *bool
	IncludeUnpublished bool
	InstructorID       *uuid.UUID
	CreatorType        string
	MineUserID         *uuid.UUID
	Page               int
	Limit              int
	ViewerRole         string
}

type CreateCourseDTO struct {
	Title          string     `json:"title" binding:"required"`
	Description    string     `json:"description"`
	Subtitle       string     `json:"subtitle"`
	Category       string     `json:"category"`
	CategoryID     *uuid.UUID `json:"categoryId"`
	Thumbnail      string     `json:"thumbnail"`
	Image          string     `json:"image"`
	Price          float64    `json:"price"`
	DiscountPrice  *float64   `json:"discountPrice"`
	IsPublished    *bool      `json:"isPublished"`
	IsFeatured     *bool      `json:"isFeatured"`
	LearningPoints []string   `json:"learningPoints"`
	Modules        []any      `json:"modules"`
}

type UpdateCourseDTO struct {
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	Subtitle       *string    `json:"subtitle"`
	Category       *string    `json:"category"`
	CategoryID     *uuid.UUID `json:"categoryId"`
	Thumbnail      *string    `json:"thumbnail"`
	Image          *string    `json:"image"`
	Price          *float64   `json:"price"`
	DiscountPrice  *float64   `json:"discountPrice"`
	IsPublished    *bool      `json:"isPublished"`
	IsFeatured     *bool      `json:"isFeatured"`
	LearningPoints []string   `json:"learningPoints"`
}

type ModuleDTO struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	IsPublished *bool  `json:"isPublished"`
}

type LessonDTO struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	VideoURL    string     `json:"videoUrl"`
	Duration    string     `json:"duration"`
	IsFree      *bool      `json:"isFree"`
	IsPublished *bool      `json:"isPublished"`
	QuizSetID   *uuid.UUID `json:"quizSetId"`
	Position    *int       `json:"position"`
}

type ResourceDTO struct {
	Title    string `json:"title" binding:"required"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Size     string `json:"size"`
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

type ReorderDTO struct {
	ModuleIDs []uuid.UUID `json:"moduleIds"`
	LessonIDs []uuid.UUID `json:"lessonIds"`
}

type ReviewDTO struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"required"`
}

type NoteDTO struct {
	Timestamp int    `json:"timestamp"`
	Text      string `json:"text" binding:"required"`
}

type Note struct {
	ID        uuid.UUID `json:"id"`
	Timestamp int       `json:"timestamp"`
	Text      string    `json:"text"`
	CreatedAt string    `json:"createdAt"`
}

type DiscussionReply struct {
	ID         uuid.UUID `json:"id"`
	UserName   string    `json:"userName"`
	UserAvatar *string   `json:"userAvatar"`
	UserRole   string    `json:"userRole"`
	Content    string    `json:"content"`
	CreatedAt  string    `json:"createdAt"`
}

type Discussion struct {
	ID         uuid.UUID          `json:"id"`
	UserName   string             `json:"userName"`
	UserAvatar *string            `json:"userAvatar"`
	UserRole   string             `json:"userRole"`
	Content    string             `json:"content"`
	CreatedAt  string             `json:"createdAt"`
	Upvotes    int                `json:"upvotes"`
	Replies    []DiscussionReply  `json:"replies"`
}

type Viewer struct {
	UserID uuid.UUID
	Role   string
}
