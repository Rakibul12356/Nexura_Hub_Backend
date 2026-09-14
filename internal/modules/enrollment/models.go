// internal/modules/enrollment/models.go
package enrollment

import (
	"time"

	"github.com/google/uuid"
	"nexura-backend/internal/modules/course"
)

type Enrollment struct {
	ID                 uuid.UUID      `json:"id"`
	StudentID          uuid.UUID      `json:"studentId"`
	CourseID           uuid.UUID      `json:"courseId"`
	EnrolledAt         time.Time      `json:"enrolledAt"`
	CompletedAt        *time.Time     `json:"completedAt"`
	ProgressPercentage float64        `json:"progressPercentage"`
	Course             *course.Course `json:"course,omitempty"`
}

type EnrollRequestDTO struct {
	PaymentMethod string `json:"paymentMethod" binding:"required"`
}

type EnrollResponseDTO struct {
	EnrollmentID        uuid.UUID  `json:"enrollmentId"`
	CourseID            uuid.UUID  `json:"courseId"`
	GroupConversationID *uuid.UUID `json:"groupConversationId"`
}

type CreatorType string

const (
	CreatorInstructor CreatorType = "instructor"
	CreatorAdmin      CreatorType = "admin"
)

type TransactionStatus string

const (
	TxCompleted TransactionStatus = "completed"
	TxPending   TransactionStatus = "pending"
	TxRefunded  TransactionStatus = "refunded"
)

type Transaction struct {
	ID                    uuid.UUID         `json:"id"`
	CourseID              uuid.UUID         `json:"courseId"`
	StudentID             uuid.UUID         `json:"studentId"`
	InstructorID          uuid.UUID         `json:"instructorId"`
	CreatorType           CreatorType       `json:"creatorType"`
	GrossAmount           float64           `json:"grossAmount"`
	AdminCommissionRate   float64           `json:"adminCommissionRate"`
	AdminCommissionAmount float64           `json:"adminCommissionAmount"`
	InstructorEarnings    float64           `json:"instructorEarnings"`
	Status                TransactionStatus `json:"status"`
	PaymentMethod         string            `json:"paymentMethod"`
	CreatedAt             time.Time         `json:"createdAt"`
}
