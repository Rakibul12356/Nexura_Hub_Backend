package chat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/utils"
)

type ChatRepository interface {
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error)
	GetConversationByID(ctx context.Context, id, userID uuid.UUID) (*Conversation, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, page, limit int) ([]ChatMessage, int, error)
	CreateMessage(ctx context.Context, msg *ChatMessage) error
	ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) ([]EmojiReaction, error)
	GetOrCreateDirect(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error)
	CreateGroup(ctx context.Context, instructorID, courseID uuid.UUID, name string) (*Conversation, error)
	JoinGroup(ctx context.Context, conversationID, userID uuid.UUID) error
	MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error
	IsMember(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]EmojiReaction, error)
}

type postgresChatRepository struct{ db *sql.DB }

func NewChatRepository(db *sql.DB) ChatRepository {
	return &postgresChatRepository{db: db}
}

func (r *postgresChatRepository) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.type, c.name, c.avatar, c.course_id, COALESCE(co.title,''), c.instructor_id, c.created_at, c.updated_at,
		       COALESCE(cm.unread_count,0)
		FROM conversations c
		JOIN conversation_members cm ON c.id = cm.conversation_id
		LEFT JOIN courses co ON co.id = c.course_id
		WHERE cm.user_id = $1
		ORDER BY c.updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Avatar, &c.CourseID, &c.CourseTitle, &c.InstructorID, &c.CreatedAt, &c.UpdatedAtRaw, &c.UnreadCount); err != nil {
			return nil, err
		}
		c.UpdatedAt = utils.DisplayTime(c.UpdatedAtRaw)
		c.Members = r.members(ctx, c.ID)
		c.LastMessage = r.lastMessage(ctx, c.ID)
		if c.Type == ChatDirect {
			c.Name = r.directName(ctx, c.ID, userID)
		}
		list = append(list, c)
	}
	if list == nil {
		list = []Conversation{}
	}
	return list, nil
}

func (r *postgresChatRepository) GetConversationByID(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	var c Conversation
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.type, c.name, c.avatar, c.course_id, COALESCE(co.title,''), c.instructor_id, c.created_at, c.updated_at,
		       COALESCE(cm.unread_count,0)
		FROM conversations c
		JOIN conversation_members cm ON c.id=cm.conversation_id AND cm.user_id=$2
		LEFT JOIN courses co ON co.id=c.course_id
		WHERE c.id=$1
	`, id, userID).Scan(&c.ID, &c.Type, &c.Name, &c.Avatar, &c.CourseID, &c.CourseTitle, &c.InstructorID, &c.CreatedAt, &c.UpdatedAtRaw, &c.UnreadCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}
	c.UpdatedAt = utils.DisplayTime(c.UpdatedAtRaw)
	c.Members = r.members(ctx, c.ID)
	c.LastMessage = r.lastMessage(ctx, c.ID)
	if c.Type == ChatDirect {
		c.Name = r.directName(ctx, c.ID, userID)
	}
	return &c, nil
}

func (r *postgresChatRepository) members(ctx context.Context, id uuid.UUID) []Member {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.first_name || ' ' || u.last_name, u.avatar, u.role
		FROM conversation_members cm JOIN users u ON u.id=cm.user_id WHERE cm.conversation_id=$1
	`, id)
	if err != nil {
		return []Member{}
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.Avatar, &m.Role); err == nil {
			out = append(out, m)
		}
	}
	if out == nil {
		return []Member{}
	}
	return out
}

func (r *postgresChatRepository) lastMessage(ctx context.Context, id uuid.UUID) *ChatMessage {
	msgs, _, err := r.GetMessages(ctx, id, 1, 1)
	if err != nil || len(msgs) == 0 {
		return nil
	}
	return &msgs[len(msgs)-1]
}

func (r *postgresChatRepository) directName(ctx context.Context, convID, userID uuid.UUID) string {
	var name string
	_ = r.db.QueryRowContext(ctx, `
		SELECT u.first_name || ' ' || u.last_name || CASE WHEN u.role='instructor' THEN ' (Instructor)' ELSE '' END
		FROM conversation_members cm JOIN users u ON u.id=cm.user_id
		WHERE cm.conversation_id=$1 AND cm.user_id <> $2 LIMIT 1
	`, convID, userID).Scan(&name)
	return name
}

func (r *postgresChatRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, page, limit int) ([]ChatMessage, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	var total int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE conversation_id=$1 AND is_deleted=false`, conversationID).Scan(&total)
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, u.first_name || ' ' || u.last_name, u.avatar, u.role,
		       COALESCE(m.content,''), m.image_url, m.reply_to_id, m.attachment_type, m.attachment_url, m.attachment_name, m.created_at
		FROM messages m JOIN users u ON u.id=m.sender_id
		WHERE m.conversation_id=$1 AND m.is_deleted=false
		ORDER BY m.created_at ASC
		LIMIT $2 OFFSET $3
	`, conversationID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []ChatMessage
	for rows.Next() {
		var m ChatMessage
		var attType, attURL, attName *string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.SenderAvatar, &m.SenderRole, &m.Content, &m.ImageURL, &m.ReplyToID, &attType, &attURL, &attName, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		m.Timestamp = utils.DisplayTime(m.CreatedAt)
		m.IsRead = true
		m.Reactions, _ = r.GetMessageReactions(ctx, m.ID)
		if m.ReplyToID != nil {
			var rt ReplyTo
			_ = r.db.QueryRowContext(ctx, `
				SELECT m.id, u.first_name, COALESCE(m.content,'') FROM messages m JOIN users u ON u.id=m.sender_id WHERE m.id=$1
			`, *m.ReplyToID).Scan(&rt.ID, &rt.SenderName, &rt.Content)
			m.ReplyTo = &rt
		}
		if attURL != nil && *attURL != "" {
			t := "file"
			if attType != nil {
				t = *attType
			}
			n := ""
			if attName != nil {
				n = *attName
			}
			m.Attachment = &Attachment{Type: t, URL: *attURL, Name: n}
		}
		out = append(out, m)
	}
	if out == nil {
		out = []ChatMessage{}
	}
	return out, total, nil
}

func (r *postgresChatRepository) GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]EmojiReaction, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT emoji, array_agg(user_id::text) FROM message_reactions WHERE message_id=$1 GROUP BY emoji`, messageID)
	if err != nil {
		return []EmojiReaction{}, err
	}
	defer rows.Close()
	var out []EmojiReaction
	for rows.Next() {
		var e EmojiReaction
		var users pq.StringArray
		if err := rows.Scan(&e.Emoji, &users); err == nil {
			e.Users = []string(users)
			e.Count = len(users)
			out = append(out, e)
		}
	}
	if out == nil {
		return []EmojiReaction{}, nil
	}
	return out, nil
}

func (r *postgresChatRepository) CreateMessage(ctx context.Context, msg *ChatMessage) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO messages (id, conversation_id, sender_id, content, image_url, reply_to_id)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at
	`, msg.ID, msg.ConversationID, msg.SenderID, msg.Content, msg.ImageURL, msg.ReplyToID).Scan(&msg.CreatedAt)
	if err != nil {
		return err
	}
	msg.Timestamp = utils.DisplayTime(msg.CreatedAt)
	msg.Reactions = []EmojiReaction{}
	_, _ = r.db.ExecContext(ctx, `UPDATE conversations SET updated_at=NOW() WHERE id=$1`, msg.ConversationID)
	_, _ = r.db.ExecContext(ctx, `UPDATE conversation_members SET unread_count = unread_count+1 WHERE conversation_id=$1 AND user_id <> $2`, msg.ConversationID, msg.SenderID)
	return nil
}

func (r *postgresChatRepository) ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) ([]EmojiReaction, error) {
	var exists bool
	_ = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3)`, messageID, userID, emoji).Scan(&exists)
	if exists {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3`, messageID, userID, emoji)
	} else {
		_, _ = r.db.ExecContext(ctx, `INSERT INTO message_reactions (message_id, user_id, emoji) VALUES ($1,$2,$3)`, messageID, userID, emoji)
	}
	return r.GetMessageReactions(ctx, messageID)
}

func (r *postgresChatRepository) GetOrCreateDirect(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error) {
	a, b := userA, userB
	if b.String() < a.String() {
		a, b = b, a
	}
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT conversation_id FROM direct_pairs WHERE user_a=$1 AND user_b=$2`, a, b).Scan(&id)
	if err == nil {
		return r.GetConversationByID(ctx, id, userA)
	}
	id = uuid.New()
	_, err = r.db.ExecContext(ctx, `INSERT INTO conversations (id, type, name) VALUES ($1,'direct','Direct')`, id)
	if err != nil {
		return nil, err
	}
	_, _ = r.db.ExecContext(ctx, `INSERT INTO direct_pairs (conversation_id, user_a, user_b) VALUES ($1,$2,$3)`, id, a, b)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2),($1,$3)`, id, userA, userB)
	return r.GetConversationByID(ctx, id, userA)
}

func (r *postgresChatRepository) CreateGroup(ctx context.Context, instructorID, courseID uuid.UUID, name string) (*Conversation, error) {
	var existing uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT id FROM conversations WHERE course_id=$1 AND type='group'`, courseID).Scan(&existing)
	if err == nil {
		return r.GetConversationByID(ctx, existing, instructorID)
	}
	var title string
	_ = r.db.QueryRowContext(ctx, `SELECT title FROM courses WHERE id=$1`, courseID).Scan(&title)
	if name == "" {
		name = title + " Community Group"
	}
	id := uuid.New()
	_, err = r.db.ExecContext(ctx, `INSERT INTO conversations (id, type, name, course_id, instructor_id) VALUES ($1,'group',$2,$3,$4)`, id, name, courseID, instructorID)
	if err != nil {
		return nil, err
	}
	_, _ = r.db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2)`, id, instructorID)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4)`,
		uuid.New(), id, instructorID, "Welcome to the official group chat for "+title+"!")
	return r.GetConversationByID(ctx, id, instructorID)
}

func (r *postgresChatRepository) JoinGroup(ctx context.Context, conversationID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, conversationID, userID)
	return err
}

func (r *postgresChatRepository) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE conversation_members SET unread_count=0, last_read_at=NOW() WHERE conversation_id=$1 AND user_id=$2`, conversationID, userID)
	return err
}

func (r *postgresChatRepository) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE messages SET is_deleted=true, content='' WHERE id=$1 AND sender_id=$2`, messageID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrMessageNotFound
	}
	return nil
}

func (r *postgresChatRepository) IsMember(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id=$2)`, conversationID, userID).Scan(&exists)
	return exists, err
}
