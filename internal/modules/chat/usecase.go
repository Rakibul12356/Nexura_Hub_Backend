// internal/modules/chat/usecase.go
package chat

import (
	"context"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/modules/auth"
)

type ChatUsecase interface {
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, cursorID *uuid.UUID, limit int) ([]ChatMessage, error)
	SendMessage(ctx context.Context, senderID uuid.UUID, dto SendMessageDTO) (*ChatMessage, error)
	ToggleReaction(ctx context.Context, userID uuid.UUID, dto SendReactionDTO) ([]EmojiReaction, error)
}

type chatUsecase struct {
	chatRepo ChatRepository
	userRepo auth.UserRepository
}

func NewChatUsecase(chatRepo ChatRepository, userRepo auth.UserRepository) ChatUsecase {
	return &chatUsecase{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

func (u *chatUsecase) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	return u.chatRepo.GetUserConversations(ctx, userID)
}

func (u *chatUsecase) GetMessages(ctx context.Context, conversationID uuid.UUID, cursorID *uuid.UUID, limit int) ([]ChatMessage, error) {
	return u.chatRepo.GetMessages(ctx, conversationID, cursorID, limit)
}

func (u *chatUsecase) SendMessage(ctx context.Context, senderID uuid.UUID, dto SendMessageDTO) (*ChatMessage, error) {
	isMember, err := u.chatRepo.IsMember(ctx, dto.ConversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, appErrors.ErrForbidden
	}

	sender, err := u.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, err
	}

	msg := &ChatMessage{
		ID:             uuid.New(),
		ConversationID: dto.ConversationID,
		SenderID:       senderID,
		SenderName:     sender.FirstName + " " + sender.LastName,
		SenderAvatar:   sender.Avatar,
		SenderRole:     sender.Role,
		Content:        dto.Content,
		ImageURL:       dto.ImageURL,
		ReplyToID:      dto.ReplyToID,
		Reactions:      []EmojiReaction{},
	}

	if err := u.chatRepo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (u *chatUsecase) ToggleReaction(ctx context.Context, userID uuid.UUID, dto SendReactionDTO) ([]EmojiReaction, error) {
	return u.chatRepo.ToggleReaction(ctx, dto.MessageID, userID, dto.Emoji)
}
