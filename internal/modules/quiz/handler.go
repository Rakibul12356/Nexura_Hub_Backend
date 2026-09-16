package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/utils"
)

type QuestionDTO struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Points      int    `json:"points"`
	Options     []struct {
		Label     string `json:"label"`
		IsCorrect bool   `json:"isCorrect"`
	} `json:"options"`
}

type QuizSet struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	TotalMarks  int        `json:"totalMarks"`
	IsPublished bool       `json:"isPublished"`
	Questions   []Question `json:"questions"`
}

type Question struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Points      int       `json:"points"`
	Options     []Option  `json:"options"`
}

type Option struct {
	ID        uuid.UUID `json:"id"`
	Label     string    `json:"label"`
	IsCorrect *bool     `json:"isCorrect,omitempty"`
}

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) List(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	q := `SELECT id, title, description, total_marks, is_published FROM quiz_sets`
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
	list := []QuizSet{}
	for rows.Next() {
		var s QuizSet
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.TotalMarks, &s.IsPublished); err != nil {
			response.Internal(c, err)
			return
		}
		s.Questions = h.questions(c, s.ID, true)
		list = append(list, s)
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *Handler) Create(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	id := uuid.New()
	_, err := h.db.ExecContext(c, `INSERT INTO quiz_sets (id, instructor_id, title, description, total_marks, is_published) VALUES ($1,$2,$3,$4,0,false)`,
		id, userID, dto.Title, nullStr(dto.Description))
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Quiz set created", gin.H{"id": id, "title": dto.Title, "questions": []any{}})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("quizId"))
	if err != nil {
		id, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	var s QuizSet
	err = h.db.QueryRowContext(c, `SELECT id, title, description, total_marks, is_published FROM quiz_sets WHERE id=$1`, id).
		Scan(&s.ID, &s.Title, &s.Description, &s.TotalMarks, &s.IsPublished)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, appErrors.ErrQuizNotFound.Error())
		return
	}
	role := middleware.CurrentRole(c)
	showCorrect := role == "admin" || role == "instructor"
	s.Questions = h.questions(c, s.ID, showCorrect)
	response.Success(c, http.StatusOK, "OK", s)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("quizId"))
	if err != nil {
		id, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsPublished *bool  `json:"isPublished"`
	}
	_ = c.ShouldBindJSON(&dto)
	_, err = h.db.ExecContext(c, `UPDATE quiz_sets SET title=COALESCE(NULLIF($1,''), title), description=COALESCE(NULLIF($2,''), description), is_published=COALESCE($3, is_published), updated_at=NOW() WHERE id=$4`,
		dto.Title, dto.Description, dto.IsPublished, id)
	if err != nil {
		response.Internal(c, err)
		return
	}
	h.Get(c)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("quizId"))
	if err != nil {
		id, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	_, err = h.db.ExecContext(c, `DELETE FROM quiz_sets WHERE id=$1`, id)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Quiz deleted", gin.H{})
}

func (h *Handler) AddQuestion(c *gin.Context) {
	quizID, err := uuid.Parse(c.Param("quizId"))
	if err != nil {
		quizID, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto QuestionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if dto.Points == 0 {
		dto.Points = 5
	}
	qid := uuid.New()
	var pos int
	_ = h.db.QueryRowContext(c, `SELECT COALESCE(MAX(position),0)+1 FROM quiz_questions WHERE quiz_set_id=$1`, quizID).Scan(&pos)
	_, err = h.db.ExecContext(c, `INSERT INTO quiz_questions (id, quiz_set_id, title, description, points, position) VALUES ($1,$2,$3,$4,$5,$6)`,
		qid, quizID, dto.Title, nullStr(dto.Description), dto.Points, pos)
	if err != nil {
		response.Internal(c, err)
		return
	}
	for i, opt := range dto.Options {
		_, _ = h.db.ExecContext(c, `INSERT INTO quiz_options (id, question_id, label, is_correct, position) VALUES ($1,$2,$3,$4,$5)`,
			uuid.New(), qid, opt.Label, opt.IsCorrect, i+1)
	}
	_, _ = h.db.ExecContext(c, `UPDATE quiz_sets SET total_marks = (SELECT COALESCE(SUM(points),0) FROM quiz_questions WHERE quiz_set_id=$1) WHERE id=$1`, quizID)
	response.Success(c, http.StatusCreated, "Question added", gin.H{"id": qid})
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	qid, err := uuid.Parse(c.Param("qId"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	quizID := c.Param("quizId")
	_, err = h.db.ExecContext(c, `DELETE FROM quiz_questions WHERE id=$1`, qid)
	if err != nil {
		response.Internal(c, err)
		return
	}
	_, _ = h.db.ExecContext(c, `UPDATE quiz_sets SET total_marks = (SELECT COALESCE(SUM(points),0) FROM quiz_questions WHERE quiz_set_id=$1) WHERE id=$1`, quizID)
	response.Success(c, http.StatusOK, "Question deleted", gin.H{})
}

func (h *Handler) Submit(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	quizID, err := uuid.Parse(c.Param("quizId"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto struct {
		LessonID string            `json:"lessonId"`
		Answers  map[string]string `json:"answers"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	questions := h.questions(c, quizID, true)
	var score, total float64
	correct := 0
	for _, q := range questions {
		total += float64(q.Points)
		ans := dto.Answers[q.ID.String()]
		for _, opt := range q.Options {
			if opt.IsCorrect != nil && *opt.IsCorrect && opt.ID.String() == ans {
				score += float64(q.Points)
				correct++
			}
		}
	}
	passed := total > 0 && (score/total)*100 >= 50
	raw, _ := json.Marshal(dto.Answers)
	var lessonID, courseID *uuid.UUID
	if dto.LessonID != "" {
		if id, err := uuid.Parse(dto.LessonID); err == nil {
			lessonID = &id
			var cid uuid.UUID
			_ = h.db.QueryRowContext(c, `SELECT m.course_id FROM lessons l JOIN modules m ON m.id=l.module_id WHERE l.id=$1`, id).Scan(&cid)
			courseID = &cid
		}
	}
	_, _ = h.db.ExecContext(c, `INSERT INTO quiz_attempts (id, quiz_set_id, lesson_id, user_id, course_id, answers, score, total, passed) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.New(), quizID, lessonID, userID, courseID, raw, score, total, passed)
	response.Success(c, http.StatusOK, "Quiz submitted", gin.H{
		"score": utils.RoundMoney(score), "total": total, "passed": passed, "correctCount": correct, "totalQuestions": len(questions),
	})
}

func (h *Handler) questions(ctx context.Context, quizID uuid.UUID, showCorrect bool) []Question {
	rows, err := h.db.QueryContext(ctx, `SELECT id, title, description, points FROM quiz_questions WHERE quiz_set_id=$1 ORDER BY position`, quizID)
	if err != nil {
		return []Question{}
	}
	defer rows.Close()
	var qs []Question
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.ID, &q.Title, &q.Description, &q.Points); err != nil {
			continue
		}
		q.Options = []Option{}
		orows, err := h.db.QueryContext(ctx, `SELECT id, label, is_correct FROM quiz_options WHERE question_id=$1 ORDER BY position`, q.ID)
		if err == nil {
			for orows.Next() {
				var o Option
				var correct bool
				if err := orows.Scan(&o.ID, &o.Label, &correct); err == nil {
					if showCorrect {
						o.IsCorrect = &correct
					}
					q.Options = append(q.Options, o)
				}
			}
			orows.Close()
		}
		qs = append(qs, q)
	}
	if qs == nil {
		return []Question{}
	}
	return qs
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
