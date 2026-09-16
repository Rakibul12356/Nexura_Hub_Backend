package certificate

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/utils"
)

type Cert struct {
	PublicID       string `json:"publicId"`
	StudentName    string `json:"studentName"`
	CourseTitle    string `json:"courseTitle"`
	CompletionDate string `json:"completionDate"`
	VerifyURL      string `json:"verifyUrl"`
}

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) Issue(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var progress float64
	var student, title string
	err = h.db.QueryRowContext(c, `
		SELECT e.progress, u.first_name || ' ' || u.last_name, c.title
		FROM enrollments e JOIN users u ON u.id=e.user_id JOIN courses c ON c.id=e.course_id
		WHERE e.user_id=$1 AND e.course_id=$2
	`, userID, courseID).Scan(&progress, &student, &title)
	if errors.Is(err, sql.ErrNoRows) {
		response.Unprocessable(c, appErrors.ErrNotEnrolled.Error())
		return
	}
	if err != nil {
		response.Internal(c, err)
		return
	}
	if progress < 100 {
		response.Unprocessable(c, appErrors.ErrCertificateNotReady.Error())
		return
	}
	publicID := utils.CertificatePublicID()
	_, err = h.db.ExecContext(c, `
		INSERT INTO certificates (id, public_id, user_id, course_id, student_name, course_title)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (user_id, course_id) DO NOTHING
	`, uuid.New(), publicID, userID, courseID, student, title)
	if err != nil {
		response.Internal(c, err)
		return
	}
	h.Me(c)
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var cert Cert
	var issued time.Time
	err = h.db.QueryRowContext(c, `SELECT public_id, student_name, course_title, issued_at FROM certificates WHERE user_id=$1 AND course_id=$2`, userID, courseID).
		Scan(&cert.PublicID, &cert.StudentName, &cert.CourseTitle, &issued)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, appErrors.ErrCertificateNotFound.Error())
		return
	}
	if err != nil {
		response.Internal(c, err)
		return
	}
	cert.CompletionDate = issued.Format("January 2, 2006")
	cert.VerifyURL = fmt.Sprintf("https://nexurahub.com/verify/%s", cert.PublicID)
	response.Success(c, http.StatusOK, "OK", cert)
}

func (h *Handler) Verify(c *gin.Context) {
	var cert Cert
	var issued time.Time
	err := h.db.QueryRowContext(c, `SELECT public_id, student_name, course_title, issued_at FROM certificates WHERE public_id=$1`, c.Param("publicId")).
		Scan(&cert.PublicID, &cert.StudentName, &cert.CourseTitle, &issued)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, appErrors.ErrCertificateNotFound.Error())
		return
	}
	if err != nil {
		response.Internal(c, err)
		return
	}
	cert.CompletionDate = issued.Format("January 2, 2006")
	cert.VerifyURL = fmt.Sprintf("https://nexurahub.com/verify/%s", cert.PublicID)
	response.Success(c, http.StatusOK, "Valid certificate", cert)
}
