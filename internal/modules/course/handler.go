// internal/modules/course/handler.go
package course

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type CourseHandler struct {
	courseUsecase CourseUsecase
}

func NewCourseHandler(courseUsecase CourseUsecase) *CourseHandler {
	return &CourseHandler{courseUsecase: courseUsecase}
}

func (h *CourseHandler) ListCourses(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	courses, total, err := h.courseUsecase.ListPublicCourses(c.Request.Context(), search, category, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"meta": gin.H{
			"total":      total,
			"page":       page,
			"totalPages": totalPages,
		},
		"data": courses,
	})
}

func (h *CourseHandler) GetCourseBySlug(c *gin.Context) {
	slug := c.Param("slug")
	course, err := h.courseUsecase.GetCourseBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, appErrors.ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   course,
	})
}

func (h *CourseHandler) GetCategories(c *gin.Context) {
	categories, err := h.courseUsecase.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   categories,
	})
}

func (h *CourseHandler) CreateCourse(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	instructorID := userIDVal.(uuid.UUID)

	var dto CreateCourseDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	course, err := h.courseUsecase.CreateCourse(c.Request.Context(), instructorID, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Course created successfully",
		"data":    course,
	})
}
