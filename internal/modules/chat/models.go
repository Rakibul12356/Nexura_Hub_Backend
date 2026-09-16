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

type Member struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Avatar   *string   `json:"avatar"`
	Role     string    `json:"role"`
	IsOnline bool      `json:"isOnline"`
}

type ReplyTo struct {
	ID         uuid.UUID `json:"id"`
	SenderName string    `json:"senderName"`
	Content    string    `json:"content"`
}

type Attachment struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

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
	CourseTitle  string       `json:"courseTitle,omitempty"`
	InstructorID *uuid.UUID   `json:"instructorId"`
	Members      []Member     `json:"members"`
	LastMessage  *ChatMessage `json:"lastMessage,omitempty"`
	UnreadCount  int          `json:"unreadCount"`
	UpdatedAt    string       `json:"updatedAt"`
	CreatedAt    time.Time    `json:"-"`
	UpdatedAtRaw time.Time    `json:"-"`
}

type ChatMessage struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversationId"`
	SenderID       uuid.UUID       `json:"senderId"`
	SenderName     string          `json:"senderName,omitempty"`
	SenderAvatar   *string         `json:"senderAvatar,omitempty"`
	SenderRole     auth.UserRole   `json:"senderRole,omitempty"`
	Content        string          `json:"content"`
	Timestamp      string          `json:"timestamp"`
	IsRead         bool            `json:"isRead"`
	ImageURL       *string         `json:"imageUrl"`
	ReplyTo        *ReplyTo        `json:"replyTo,omitempty"`
	Attachment     *Attachment     `json:"attachment,omitempty"`
	Reactions      []EmojiReaction `json:"reactions"`
	ReplyToID      *uuid.UUID      `json:"-"`
	CreatedAt      time.Time       `json:"createdAt,omitempty"`
}

type SendMessageDTO struct {
	ConversationID uuid.UUID  `json:"conversationId"`
	Content        string     `json:"content"`
	ImageURL       *string    `json:"imageUrl"`
	ReplyToID      *uuid.UUID `json:"replyToId"`
}

type SendReactionDTO struct {
	ConversationID uuid.UUID `json:"conversationId"`
	MessageID      uuid.UUID `json:"messageId"`
	Emoji          string    `json:"emoji"`
	UserID         uuid.UUID `json:"userId"`
}
