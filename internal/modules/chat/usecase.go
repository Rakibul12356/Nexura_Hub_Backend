package chat

import (
	"context"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/modules/auth"
)

type ChatUsecase interface {
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error)
	GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, page, limit int) ([]ChatMessage, int, error)
	SendMessage(ctx context.Context, senderID uuid.UUID, dto SendMessageDTO) (*ChatMessage, error)
	ToggleReaction(ctx context.Context, userID uuid.UUID, dto SendReactionDTO) ([]EmojiReaction, error)
	GetOrCreateDirect(ctx context.Context, userID, other uuid.UUID) (*Conversation, error)
	CreateGroup(ctx context.Context, instructorID, courseID uuid.UUID, name string) (*Conversation, error)
	JoinGroup(ctx context.Context, conversationID, userID uuid.UUID) error
	MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error
}

type chatUsecase struct {
	chatRepo ChatRepository
	userRepo auth.UserRepository
}

func NewChatUsecase(chatRepo ChatRepository, userRepo auth.UserRepository) ChatUsecase {
	return &chatUsecase{chatRepo: chatRepo, userRepo: userRepo}
}

func (u *chatUsecase) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	return u.chatRepo.GetUserConversations(ctx, userID)
}

func (u *chatUsecase) GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	return u.chatRepo.GetConversationByID(ctx, id, userID)
}

func (u *chatUsecase) GetMessages(ctx context.Context, conversationID uuid.UUID, page, limit int) ([]ChatMessage, int, error) {
	return u.chatRepo.GetMessages(ctx, conversationID, page, limit)
}

func (u *chatUsecase) SendMessage(ctx context.Context, senderID uuid.UUID, dto SendMessageDTO) (*ChatMessage, error) {
	ok, err := u.chatRepo.IsMember(ctx, dto.ConversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, appErrors.ErrForbidden
	}
	sender, err := u.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, err
	}
	msg := &ChatMessage{
		ID: uuid.New(), ConversationID: dto.ConversationID, SenderID: senderID,
		SenderName: sender.FirstName + " " + sender.LastName, SenderAvatar: sender.Avatar,
		SenderRole: sender.Role, Content: dto.Content, ImageURL: dto.ImageURL, ReplyToID: dto.ReplyToID,
	}
	if err := u.chatRepo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (u *chatUsecase) ToggleReaction(ctx context.Context, userID uuid.UUID, dto SendReactionDTO) ([]EmojiReaction, error) {
	return u.chatRepo.ToggleReaction(ctx, dto.MessageID, userID, dto.Emoji)
}

func (u *chatUsecase) GetOrCreateDirect(ctx context.Context, userID, other uuid.UUID) (*Conversation, error) {
	return u.chatRepo.GetOrCreateDirect(ctx, userID, other)
}

func (u *chatUsecase) CreateGroup(ctx context.Context, instructorID, courseID uuid.UUID, name string) (*Conversation, error) {
	return u.chatRepo.CreateGroup(ctx, instructorID, courseID, name)
}

func (u *chatUsecase) JoinGroup(ctx context.Context, conversationID, userID uuid.UUID) error {
	return u.chatRepo.JoinGroup(ctx, conversationID, userID)
}

func (u *chatUsecase) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	return u.chatRepo.MarkRead(ctx, conversationID, userID)
}

func (u *chatUsecase) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	return u.chatRepo.DeleteMessage(ctx, messageID, userID)
}
