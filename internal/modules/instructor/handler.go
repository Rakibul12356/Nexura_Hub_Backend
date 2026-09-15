// internal/modules/instructor/handler.go
package instructor

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InstructorHandler struct {
	instructorUsecase InstructorUsecase
}

func NewInstructorHandler(instructorUsecase InstructorUsecase) *InstructorHandler {
	return &InstructorHandler{instructorUsecase: instructorUsecase}
}

func (h *InstructorHandler) GetDashboardStats(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)

	stats, err := h.instructorUsecase.GetDashboardStats(c.Request.Context(), instructorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

func (h *InstructorHandler) GetDashboardCourses(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)
	data, err := h.instructorUsecase.GetInstructorCourses(c.Request.Context(), instructorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": data})
}

func (h *InstructorHandler) GetDashboardLives(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)
	data, err := h.instructorUsecase.GetInstructorLives(c.Request.Context(), instructorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": data})
}

func (h *InstructorHandler) GetDashboardQuizSets(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)
	data, err := h.instructorUsecase.GetInstructorQuizSets(c.Request.Context(), instructorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": data})
}

func (h *InstructorHandler) GetDashboardEnrollments(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)
	data, err := h.instructorUsecase.GetInstructorEnrollments(c.Request.Context(), instructorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": data})
}
