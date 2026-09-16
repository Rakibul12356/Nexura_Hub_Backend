package notification

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/utils"
)

type Item struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp string    `json:"timestamp"`
	IsRead    bool      `json:"isRead"`
	Link      *string   `json:"link,omitempty"`
}

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) List(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	rows, err := h.db.QueryContext(c, `SELECT id, type, title, message, is_read, link, created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		response.Success(c, http.StatusOK, "OK", []Item{})
		return
	}
	defer rows.Close()
	list := []Item{}
	for rows.Next() {
		var n Item
		var created sql.NullTime
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.IsRead, &n.Link, &created); err != nil {
			response.Internal(c, err)
			return
		}
		if created.Valid {
			n.Timestamp = utils.RelativeTime(created.Time)
		}
		list = append(list, n)
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *Handler) Read(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	_, _ = h.db.ExecContext(c, `UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2`, id, userID)
	response.Success(c, http.StatusOK, "OK", gin.H{"isRead": true})
}

func (h *Handler) ReadAll(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	_, _ = h.db.ExecContext(c, `UPDATE notifications SET is_read=true WHERE user_id=$1`, userID)
	response.Success(c, http.StatusOK, "All notifications marked read", gin.H{})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var n int
	_ = h.db.QueryRowContext(c, `SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=false`, userID).Scan(&n)
	response.Success(c, http.StatusOK, "OK", gin.H{"count": n})
}
