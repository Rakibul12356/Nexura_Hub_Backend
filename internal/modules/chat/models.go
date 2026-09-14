// internal/modules/chat/models.go
package chat

import (
	"time"

	"github.com/google/uuid"
	"nexura-backend/internal/modules/auth"
)

type ChatType string

const (
	ChatDirect ChatType = "direct"
	ChatGroup  ChatType = "group"
)

type EmojiReaction struct {
	Emoji string   `json:"emoji"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

type Conversation struct {
	ID           uuid.UUID    `json:"id"`
	Type         ChatType     `json:"type"`
	Name         string       `json:"name"`
	Avatar       *string      `json:"avatar"`
	CourseID     *uuid.UUID   `json:"courseId"`
	InstructorID *uuid.UUID   `json:"instructorId"`
	LastMessage  *ChatMessage `json:"lastMessage,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type ChatMessage struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversationId"`
	SenderID       uuid.UUID       `json:"senderId"`
	SenderName     string          `json:"senderName,omitempty"`
	SenderAvatar   *string         `json:"senderAvatar,omitempty"`
	SenderRole     auth.UserRole   `json:"senderRole,omitempty"`
	Content        *string         `json:"content"`
	ImageURL       *string         `json:"imageUrl"`
	ReplyToID      *uuid.UUID      `json:"replyToId"`
	Reactions      []EmojiReaction `json:"reactions"`
	Timestamp      string          `json:"timestamp"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type SendMessageDTO struct {
	ConversationID uuid.UUID  `json:"conversationId"`
	Content        *string    `json:"content"`
	ImageURL       *string    `json:"imageUrl"`
	ReplyToID      *uuid.UUID `json:"replyToId"`
}

type SendReactionDTO struct {
	ConversationID uuid.UUID `json:"conversationId"`
	MessageID      uuid.UUID `json:"messageId"`
	Emoji          string    `json:"emoji"`
}
