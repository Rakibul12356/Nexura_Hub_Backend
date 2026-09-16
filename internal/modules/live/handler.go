package live

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
)

type LiveClass struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Date        string     `json:"date"`
	Time        string     `json:"time"`
	Duration    *string    `json:"duration"`
	MeetingLink *string    `json:"meetingLink"`
	IsCompleted bool       `json:"isCompleted"`
	CourseID    *uuid.UUID `json:"courseId,omitempty"`
}

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) List(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	q := `SELECT id, title, description, date, time, duration, meeting_link, is_completed, course_id FROM live_classes`
	args := []any{}
	if role != "admin" {
		q += ` WHERE instructor_id=$1`
		args = append(args, userID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.db.QueryContext(c, q, args...)
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer rows.Close()
	list := []LiveClass{}
	for rows.Next() {
		var l LiveClass
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.Date, &l.Time, &l.Duration, &l.MeetingLink, &l.IsCompleted, &l.CourseID); err != nil {
			response.Internal(c, err)
			return
		}
		list = append(list, l)
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *Handler) Create(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Date        string `json:"date" binding:"required"`
		Time        string `json:"time" binding:"required"`
		MeetingLink string `json:"meetingLink"`
		CourseID    string `json:"courseId"`
		Duration    string `json:"duration"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	id := uuid.New()
	var courseID *uuid.UUID
	if dto.CourseID != "" {
		if parsed, err := uuid.Parse(dto.CourseID); err == nil {
			courseID = &parsed
		}
	}
	_, err := h.db.ExecContext(c, `INSERT INTO live_classes (id, instructor_id, course_id, title, description, date, time, duration, meeting_link) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, userID, courseID, dto.Title, null(dto.Description), dto.Date, dto.Time, null(dto.Duration), null(dto.MeetingLink))
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Live class created", gin.H{"id": id, "title": dto.Title, "date": dto.Date, "time": dto.Time, "meetingLink": dto.MeetingLink, "isCompleted": false})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var l LiveClass
	err = h.db.QueryRowContext(c, `SELECT id, title, description, date, time, duration, meeting_link, is_completed, course_id FROM live_classes WHERE id=$1`, id).
		Scan(&l.ID, &l.Title, &l.Description, &l.Date, &l.Time, &l.Duration, &l.MeetingLink, &l.IsCompleted, &l.CourseID)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, appErrors.ErrLiveNotFound.Error())
		return
	}
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", l)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto LiveClass
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	_, err = h.db.ExecContext(c, `UPDATE live_classes SET title=$1, description=$2, date=$3, time=$4, duration=$5, meeting_link=$6, is_completed=$7, updated_at=NOW() WHERE id=$8`,
		dto.Title, dto.Description, dto.Date, dto.Time, dto.Duration, dto.MeetingLink, dto.IsCompleted, id)
	if err != nil {
		response.Internal(c, err)
		return
	}
	h.Get(c)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	_, err = h.db.ExecContext(c, `DELETE FROM live_classes WHERE id=$1`, id)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Live class deleted", gin.H{})
}

func null(s string) any {
	if s == "" {
		return nil
	}
	return s
}
