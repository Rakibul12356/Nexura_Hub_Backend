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

func (h *CourseHandler) GetCourseByIDOrSlug(c *gin.Context) {
	param := c.Param("id")
	courseID, err := uuid.Parse(param)
	if err == nil {
		course, err := h.courseUsecase.GetCourseByID(c.Request.Context(), courseID)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": course})
			return
		}
	}
	course, err := h.courseUsecase.GetCourseBySlug(c.Request.Context(), param)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Course not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": course})
}

func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid course ID"})
		return
	}
	var dto CreateCourseDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	course, err := h.courseUsecase.UpdateCourse(c.Request.Context(), id, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Course updated successfully", "data": course})
}

func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid course ID"})
		return
	}
	if err := h.courseUsecase.DeleteCourse(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Course deleted successfully"})
}

func (h *CourseHandler) TogglePublish(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid course ID"})
		return
	}
	var req struct {
		IsPublished *bool `json:"isPublished"`
	}
	_ = c.ShouldBindJSON(&req)
	isPub := true
	if req.IsPublished != nil {
		isPub = *req.IsPublished
	}
	if err := h.courseUsecase.TogglePublish(c.Request.Context(), id, isPub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Publish status updated", "data": gin.H{"isPublished": isPub}})
}

func (h *CourseHandler) AddModule(c *gin.Context) {
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid course ID"})
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	mod, err := h.courseUsecase.AddModule(c.Request.Context(), courseID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": mod})
}

func (h *CourseHandler) UpdateModule(c *gin.Context) {
	moduleIDStr := c.Param("id")
	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid module ID"})
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	if err := h.courseUsecase.UpdateModule(c.Request.Context(), moduleID, req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Module updated successfully"})
}

func (h *CourseHandler) DeleteModule(c *gin.Context) {
	moduleIDStr := c.Param("id")
	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid module ID"})
		return
	}
	if err := h.courseUsecase.DeleteModule(c.Request.Context(), moduleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Module deleted successfully"})
}

func (h *CourseHandler) AddLesson(c *gin.Context) {
	moduleIDStr := c.Param("id")
	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid module ID"})
		return
	}
	var req struct {
		Title    string `json:"title" binding:"required"`
		VideoURL string `json:"videoUrl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	les, err := h.courseUsecase.AddLesson(c.Request.Context(), moduleID, req.Title, req.VideoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": les})
}

func (h *CourseHandler) UpdateLesson(c *gin.Context) {
	lessonIDStr := c.Param("id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid lesson ID"})
		return
	}
	var req struct {
		Title    string `json:"title"`
		VideoURL string `json:"videoUrl"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.courseUsecase.UpdateLesson(c.Request.Context(), lessonID, req.Title, req.VideoURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Lesson updated successfully"})
}

func (h *CourseHandler) DeleteLesson(c *gin.Context) {
	lessonIDStr := c.Param("id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid lesson ID"})
		return
	}
	if err := h.courseUsecase.DeleteLesson(c.Request.Context(), lessonID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Lesson deleted successfully"})
}
