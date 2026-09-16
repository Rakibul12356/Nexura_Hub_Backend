package instructor

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/payment"
)

type InstructorHandler struct {
	db         *sql.DB
	courseRepo course.CourseRepository
	payRepo    payment.Repository
}

func NewInstructorHandler(db *sql.DB, courseRepo course.CourseRepository, payRepo payment.Repository) *InstructorHandler {
	return &InstructorHandler{db: db, courseRepo: courseRepo, payRepo: payRepo}
}

func (h *InstructorHandler) GetDashboardStats(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	ctx := c.Request.Context()
	s := InstructorStats{}
	if role == "admin" {
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`).Scan(&s.TotalCourses)
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments`).Scan(&s.TotalEnrollments)
		_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE status='completed'`).Scan(&s.TotalRevenue, &s.InstructorEarnings)
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT user_id) FROM enrollments`).Scan(&s.TotalStudents)
	} else {
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE instructor_id=$1 AND deleted_at IS NULL`, id).Scan(&s.TotalCourses)
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments e JOIN courses c ON c.id=e.course_id WHERE c.instructor_id=$1`, id).Scan(&s.TotalEnrollments)
		_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE instructor_id=$1 AND status='completed'`, id).Scan(&s.TotalRevenue, &s.InstructorEarnings)
		_ = h.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses c ON c.id=e.course_id WHERE c.instructor_id=$1`, id).Scan(&s.TotalStudents)
	}
	response.Success(c, http.StatusOK, "OK", s)
}

func (h *InstructorHandler) GetDashboardCourses(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	isAdmin := middleware.CurrentRole(c) == "admin"
	list, err := h.courseRepo.ListInstructorCourses(c.Request.Context(), id, isAdmin)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *InstructorHandler) GetDashboardLives(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	q := `SELECT id, title, description, date, time, duration, meeting_link, is_completed FROM live_classes`
	args := []any{}
	if role != "admin" {
		q += ` WHERE instructor_id=$1`
		args = append(args, id)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.db.QueryContext(c, q, args...)
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var title, date, timeStr string
		var desc, dur, link *string
		var done bool
		if err := rows.Scan(&id, &title, &desc, &date, &timeStr, &dur, &link, &done); err == nil {
			list = append(list, gin.H{"id": id, "title": title, "description": desc, "date": date, "time": timeStr, "duration": dur, "meetingLink": link, "isCompleted": done})
		}
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *InstructorHandler) GetDashboardQuizSets(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	q := `SELECT id, title, description, total_marks, is_published, (SELECT COUNT(*) FROM quiz_questions WHERE quiz_set_id=quiz_sets.id) FROM quiz_sets`
	args := []any{}
	if role != "admin" {
		q += ` WHERE instructor_id=$1`
		args = append(args, id)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.db.QueryContext(c, q, args...)
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var qid uuid.UUID
		var title string
		var desc *string
		var marks, qcount int
		var pub bool
		if err := rows.Scan(&qid, &title, &desc, &marks, &pub, &qcount); err == nil {
			list = append(list, gin.H{"id": qid, "title": title, "description": desc, "totalMarks": marks, "isPublished": pub, "questions": make([]any, qcount)})
		}
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *InstructorHandler) GetDashboardEnrollments(c *gin.Context) {
	id, _ := middleware.CurrentUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.payRepo.ListInstructorEnrollments(c.Request.Context(), id, limit)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}
