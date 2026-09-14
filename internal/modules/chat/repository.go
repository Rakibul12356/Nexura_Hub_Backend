// internal/modules/chat/repository.go
package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type ChatRepository interface {
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error)
	GetConversationByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, cursorID *uuid.UUID, limit int) ([]ChatMessage, error)
	CreateMessage(ctx context.Context, msg *ChatMessage) error
	ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) ([]EmojiReaction, error)
	CreateConversation(ctx context.Context, conv *Conversation, memberIDs []uuid.UUID) error
	IsMember(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
}

type postgresChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) ChatRepository {
	return &postgresChatRepository{db: db}
}

func (r *postgresChatRepository) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	query := `
		SELECT c.id, c.type, c.name, c.avatar, c.course_id, c.instructor_id, c.created_at, c.updated_at
		FROM conversations c
		JOIN conversation_members cm ON c.id = cm.conversation_id
		WHERE cm.user_id = $1
		ORDER BY c.updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Avatar, &c.CourseID, &c.InstructorID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}

		lastMsgQuery := `
			SELECT m.id, m.conversation_id, m.sender_id, u.first_name || ' ' || u.last_name, u.avatar, u.role,
			       m.content, m.image_url, m.reply_to_id, COALESCE(m.reactions, '[]'::jsonb), m.created_at
			FROM chat_messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.conversation_id = $1
			ORDER BY m.created_at DESC
			LIMIT 1
		`
		var msg ChatMessage
		var reactionsJSON []byte
		err := r.db.QueryRowContext(ctx, lastMsgQuery, c.ID).Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.SenderName, &msg.SenderAvatar, &msg.SenderRole,
			&msg.Content, &msg.ImageURL, &msg.ReplyToID, &reactionsJSON, &msg.CreatedAt,
		)
		if err == nil {
			_ = json.Unmarshal(reactionsJSON, &msg.Reactions)
			msg.Timestamp = msg.CreatedAt.Format("03:04 PM")
			c.LastMessage = &msg
		}

		conversations = append(conversations, c)
	}

	return conversations, nil
}

func (r *postgresChatRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*Conversation, error) {
	query := `SELECT id, type, name, avatar, course_id, instructor_id, created_at, updated_at FROM conversations WHERE id = $1`
	var c Conversation
	err := r.db.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.Type, &c.Name, &c.Avatar, &c.CourseID, &c.InstructorID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrConversationNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *postgresChatRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, cursorID *uuid.UUID, limit int) ([]ChatMessage, error) {
	if limit < 1 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if cursorID != nil && *cursorID != uuid.Nil {
		query := `
			SELECT m.id, m.conversation_id, m.sender_id, u.first_name || ' ' || u.last_name, u.avatar, u.role,
			       m.content, m.image_url, m.reply_to_id, COALESCE(m.reactions, '[]'::jsonb), m.created_at
			FROM chat_messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.conversation_id = $1 AND m.created_at < (SELECT created_at FROM chat_messages WHERE id = $2)
			ORDER BY m.created_at DESC
			LIMIT $3
		`
		rows, err = r.db.QueryContext(ctx, query, conversationID, *cursorID, limit)
	} else {
		query := `
			SELECT m.id, m.conversation_id, m.sender_id, u.first_name || ' ' || u.last_name, u.avatar, u.role,
			       m.content, m.image_url, m.reply_to_id, COALESCE(m.reactions, '[]'::jsonb), m.created_at
			FROM chat_messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.conversation_id = $1
			ORDER BY m.created_at DESC
			LIMIT $2
		`
		rows, err = r.db.QueryContext(ctx, query, conversationID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var reactionsJSON []byte
		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.SenderName, &msg.SenderAvatar, &msg.SenderRole,
			&msg.Content, &msg.ImageURL, &msg.ReplyToID, &reactionsJSON, &msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(reactionsJSON, &msg.Reactions)
		if msg.Reactions == nil {
			msg.Reactions = []EmojiReaction{}
		}
		msg.Timestamp = msg.CreatedAt.Format("03:04 PM")
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *postgresChatRepository) CreateMessage(ctx context.Context, msg *ChatMessage) error {
	query := `
		INSERT INTO chat_messages (id, conversation_id, sender_id, content, image_url, reply_to_id, reactions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	reactionsJSON, _ := json.Marshal(msg.Reactions)
	err := r.db.QueryRowContext(ctx, query, msg.ID, msg.ConversationID, msg.SenderID, msg.Content, msg.ImageURL, msg.ReplyToID, reactionsJSON).Scan(&msg.CreatedAt)
	if err != nil {
		return err
	}

	_, _ = r.db.ExecContext(ctx, `UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`, msg.ConversationID)

	msg.Timestamp = msg.CreatedAt.Format("03:04 PM")
	return nil
}

func (r *postgresChatRepository) ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) ([]EmojiReaction, error) {
	var reactionsJSON []byte
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(reactions, '[]'::jsonb) FROM chat_messages WHERE id = $1`, messageID).Scan(&reactionsJSON)
	if err != nil {
		return nil, err
	}

	var reactions []EmojiReaction
	_ = json.Unmarshal(reactionsJSON, &reactions)

	userIDStr := userID.String()
	foundEmoji := false

	for i, rItem := range reactions {
		if rItem.Emoji == emoji {
			foundEmoji = true
			userIndex := -1
			for uIdx, uId := range rItem.Users {
				if uId == userIDStr {
					userIndex = uIdx
					break
				}
			}
			if userIndex != -1 {
				reactions[i].Users = append(reactions[i].Users[:userIndex], reactions[i].Users[userIndex+1:]...)
				reactions[i].Count--
			} else {
				reactions[i].Users = append(reactions[i].Users, userIDStr)
				reactions[i].Count++
			}
			break
		}
	}

	if !foundEmoji {
		reactions = append(reactions, EmojiReaction{
			Emoji: emoji,
			Count: 1,
			Users: []string{userIDStr},
		})
	}

	cleaned := []EmojiReaction{}
	for _, rItem := range reactions {
		if rItem.Count > 0 {
			cleaned = append(cleaned, rItem)
		}
	}

	newJSON, _ := json.Marshal(cleaned)
	_, err = r.db.ExecContext(ctx, `UPDATE chat_messages SET reactions = $1 WHERE id = $2`, newJSON, messageID)
	if err != nil {
		return nil, err
	}

	return cleaned, nil
}

func (r *postgresChatRepository) CreateConversation(ctx context.Context, conv *Conversation, memberIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if conv.ID == uuid.Nil {
		conv.ID = uuid.New()
	}

	query := `
		INSERT INTO conversations (id, type, name, avatar, course_id, instructor_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, query, conv.ID, conv.Type, conv.Name, conv.Avatar, conv.CourseID, conv.InstructorID).Scan(&conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, conv.ID, memberID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *postgresChatRepository) IsMember(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, conversationID, userID).Scan(&exists)
	return exists, err
}
