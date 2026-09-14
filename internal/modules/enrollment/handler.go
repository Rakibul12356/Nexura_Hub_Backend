// internal/modules/enrollment/handler.go
package enrollment

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type EnrollmentHandler struct {
	enrollmentUsecase EnrollmentUsecase
}

func NewEnrollmentHandler(enrollmentUsecase EnrollmentUsecase) *EnrollmentHandler {
	return &EnrollmentHandler{enrollmentUsecase: enrollmentUsecase}
}

func (h *EnrollmentHandler) EnrollCourse(c *gin.Context) {
	studentIDVal, _ := c.Get("userID")
	studentID := studentIDVal.(uuid.UUID)

	courseIDStr := c.Param("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid course ID"})
		return
	}

	var dto EnrollRequestDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	res, err := h.enrollmentUsecase.EnrollStudent(c.Request.Context(), studentID, courseID, dto.PaymentMethod)
	if err != nil {
		if errors.Is(err, appErrors.ErrAlreadyEnrolled) {
			c.JSON(http.StatusConflict, gin.H{"status": "error", "message": err.Error()})
			return
		}
		if errors.Is(err, appErrors.ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Enrolled successfully",
		"data":    res,
	})
}

func (h *EnrollmentHandler) GetMyEnrollments(c *gin.Context) {
	studentIDVal, _ := c.Get("userID")
	studentID := studentIDVal.(uuid.UUID)

	enrollments, err := h.enrollmentUsecase.GetStudentEnrollments(c.Request.Context(), studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   enrollments,
	})
}

func (h *EnrollmentHandler) CompleteLesson(c *gin.Context) {
	studentIDVal, _ := c.Get("userID")
	studentID := studentIDVal.(uuid.UUID)

	lessonIDStr := c.Param("lessonId")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid lesson ID"})
		return
	}

	if err := h.enrollmentUsecase.CompleteLesson(c.Request.Context(), studentID, lessonID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Lesson marked as completed",
	})
}
