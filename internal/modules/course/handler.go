package course

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/utils"
)

type CourseHandler struct {
	courseUsecase CourseUsecase
}

func NewCourseHandler(courseUsecase CourseUsecase) *CourseHandler {
	return &CourseHandler{courseUsecase: courseUsecase}
}

func viewerFrom(c *gin.Context) Viewer {
	id, _ := middleware.CurrentUserID(c)
	return Viewer{UserID: id, Role: middleware.CurrentRole(c)}
}

func parseUUIDParam(c *gin.Context, names ...string) (uuid.UUID, error) {
	var last error
	for _, name := range names {
		if v := c.Param(name); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				return id, nil
			}
			last = err
		}
	}
	if last != nil {
		return uuid.Nil, last
	}
	return uuid.Nil, errors.New("missing id")
}

func (h *CourseHandler) ListCourses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	v := viewerFrom(c)
	f := ListFilter{
		Search: c.Query("search"), Category: c.Query("category"), Price: c.Query("price"),
		Sort: c.Query("sort"), CreatorType: c.Query("creatorType"), Page: page, Limit: limit, ViewerRole: v.Role,
	}
	if c.Query("isFeatured") == "true" {
		t := true
		f.IsFeatured = &t
	}
	if c.Query("isPublished") == "true" {
		t := true
		f.IsPublished = &t
	}
	if c.Query("isPublished") == "false" {
		t := false
		f.IsPublished = &t
	}
	if c.Query("includeUnpublished") == "true" && v.Role == "admin" {
		f.IncludeUnpublished = true
	}
	if v.Role == "admin" {
		f.IncludeUnpublished = true
	}
	if c.Query("mine") == "true" && v.UserID != uuid.Nil {
		f.MineUserID = &v.UserID
		f.IncludeUnpublished = true
	}
	if iid := c.Query("instructorId"); iid != "" {
		if id, err := uuid.Parse(iid); err == nil {
			f.InstructorID = &id
		}
	}
	if strings.Contains(c.FullPath(), "/instructors/:id/courses") {
		if id, err := uuid.Parse(c.Param("id")); err == nil {
			f.InstructorID = &id
			t := true
			f.IsPublished = &t
		}
	}
	courses, total, err := h.courseUsecase.ListCourses(c.Request.Context(), f)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "OK", courses, utils.PageMeta(page, limit, total))
}

func (h *CourseHandler) GetCourseByIDOrSlug(c *gin.Context) {
	course, err := h.courseUsecase.GetCourse(c.Request.Context(), c.Param("id"), viewerFrom(c))
	if err != nil {
		if errors.Is(err, appErrors.ErrCourseNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", course)
}

func (h *CourseHandler) GetCategories(c *gin.Context) {
	cats, err := h.courseUsecase.GetCategories(c.Request.Context())
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", cats)
}

func (h *CourseHandler) GetCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	cat, err := h.courseUsecase.GetCategory(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "OK", cat)
}

func (h *CourseHandler) CreateCategory(c *gin.Context) {
	var dto Category
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	cat, err := h.courseUsecase.CreateCategory(c.Request.Context(), dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Category created", cat)
}

func (h *CourseHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto Category
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	cat, err := h.courseUsecase.UpdateCategory(c.Request.Context(), id, dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Category updated", cat)
}

func (h *CourseHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.courseUsecase.DeleteCategory(c.Request.Context(), id); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Category deleted", gin.H{})
}

func (h *CourseHandler) CreateCourse(c *gin.Context) {
	v := viewerFrom(c)
	var dto CreateCourseDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	course, err := h.courseUsecase.CreateCourse(c.Request.Context(), v.UserID, v.Role, dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Course created successfully", course)
}

func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto UpdateCourseDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	course, err := h.courseUsecase.UpdateCourse(c.Request.Context(), id, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Course updated successfully", course)
}

func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteCourse(c.Request.Context(), id, viewerFrom(c)); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Course deleted successfully", gin.H{})
}

func (h *CourseHandler) TogglePublish(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	published := true
	if c.Request.URL.Path != "" && (c.FullPath() == "/api/v1/courses/:id/unpublish" || c.Param("action") == "unpublish") {
		published = false
	}
	var req struct {
		IsPublished *bool `json:"isPublished"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.IsPublished != nil {
		published = *req.IsPublished
	}
	if err := h.courseUsecase.TogglePublish(c.Request.Context(), id, viewerFrom(c), published); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Publish status updated", gin.H{"isPublished": published})
}

func (h *CourseHandler) Unpublish(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.TogglePublish(c.Request.Context(), id, viewerFrom(c), false); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Course unpublished", gin.H{"isPublished": false})
}

func (h *CourseHandler) AddModule(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		courseID, err = uuid.Parse(c.Param("courseId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ModuleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	mod, err := h.courseUsecase.AddModule(c.Request.Context(), courseID, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Module created", mod)
}

func (h *CourseHandler) ListModules(c *gin.Context) {
	courseID, err := parseUUIDParam(c, "id", "courseId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	mods, err := h.courseUsecase.GetModules(c.Request.Context(), courseID, viewerFrom(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", mods)
}

func (h *CourseHandler) GetModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		id, err = uuid.Parse(c.Param("moduleId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid module ID", "VALIDATION_ERROR", nil)
		return
	}
	mod, err := h.courseUsecase.GetModule(c.Request.Context(), id, viewerFrom(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", mod)
}

func (h *CourseHandler) UpdateModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		id, err = uuid.Parse(c.Param("moduleId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid module ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ModuleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	mod, err := h.courseUsecase.UpdateModule(c.Request.Context(), id, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Module updated successfully", mod)
}

func (h *CourseHandler) DeleteModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		id, err = uuid.Parse(c.Param("moduleId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid module ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteModule(c.Request.Context(), id, viewerFrom(c)); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Module deleted successfully", gin.H{})
}

func (h *CourseHandler) ReorderModules(c *gin.Context) {
	courseID, err := parseUUIDParam(c, "id", "courseId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid course ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ReorderDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.courseUsecase.ReorderModules(c.Request.Context(), courseID, viewerFrom(c), dto.ModuleIDs); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Modules reordered", gin.H{})
}

func (h *CourseHandler) AddLesson(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		moduleID, err = uuid.Parse(c.Param("moduleId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid module ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto LessonDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	les, err := h.courseUsecase.AddLesson(c.Request.Context(), moduleID, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Lesson created", les)
}

func (h *CourseHandler) GetLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("lessonId"))
	if err != nil {
		id, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	les, err := h.courseUsecase.GetLesson(c.Request.Context(), id, viewerFrom(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", les)
}

func (h *CourseHandler) UpdateLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		id, err = uuid.Parse(c.Param("lessonId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto LessonDTO
	_ = c.ShouldBindJSON(&dto)
	les, err := h.courseUsecase.UpdateLesson(c.Request.Context(), id, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lesson updated successfully", les)
}

func (h *CourseHandler) DeleteLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		id, err = uuid.Parse(c.Param("lessonId"))
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteLesson(c.Request.Context(), id, viewerFrom(c)); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lesson deleted successfully", gin.H{})
}

func (h *CourseHandler) ReorderLessons(c *gin.Context) {
	moduleID, err := parseUUIDParam(c, "id", "moduleId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid module ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ReorderDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.courseUsecase.ReorderLessons(c.Request.Context(), moduleID, viewerFrom(c), dto.LessonIDs); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lessons reordered", gin.H{})
}

func (h *CourseHandler) AddResource(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ResourceDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.courseUsecase.AddResource(c.Request.Context(), lessonID, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Resource added", res)
}

func (h *CourseHandler) DeleteResource(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	rid, err := uuid.Parse(c.Param("resourceId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid resource ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteResource(c.Request.Context(), lessonID, rid, viewerFrom(c)); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Resource deleted", gin.H{})
}

func (h *CourseHandler) GetReviews(c *gin.Context) {
	revs, err := h.courseUsecase.GetReviews(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", revs)
}

func (h *CourseHandler) AddReview(c *gin.Context) {
	v := viewerFrom(c)
	var dto ReviewDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	rev, err := h.courseUsecase.AddReview(c.Request.Context(), c.Param("id"), v.UserID, dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Review submitted", rev)
}

func (h *CourseHandler) UpdateReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid review ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto ReviewDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	rev, err := h.courseUsecase.UpdateReview(c.Request.Context(), id, viewerFrom(c), dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Review updated", rev)
}

func (h *CourseHandler) DeleteReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid review ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteReview(c.Request.Context(), id, viewerFrom(c)); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Review deleted", gin.H{})
}

func (h *CourseHandler) GetNotes(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	notes, err := h.courseUsecase.GetNotes(c.Request.Context(), lessonID, viewerFrom(c).UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", notes)
}

func (h *CourseHandler) AddNote(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	var dto NoteDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	note, err := h.courseUsecase.AddNote(c.Request.Context(), lessonID, viewerFrom(c).UserID, dto)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Note saved", note)
}

func (h *CourseHandler) DeleteNote(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	noteID, err := uuid.Parse(c.Param("noteId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid note ID", "VALIDATION_ERROR", nil)
		return
	}
	if err := h.courseUsecase.DeleteNote(c.Request.Context(), lessonID, noteID, viewerFrom(c).UserID); err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Note deleted", gin.H{})
}

func (h *CourseHandler) GetDiscussions(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	list, err := h.courseUsecase.GetDiscussions(c.Request.Context(), lessonID, viewerFrom(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *CourseHandler) AddDiscussion(c *gin.Context) {
	lessonID, err := parseUUIDParam(c, "id", "lessonId")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid lesson ID", "VALIDATION_ERROR", nil)
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	d, err := h.courseUsecase.AddDiscussion(c.Request.Context(), lessonID, viewerFrom(c), req.Content)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Question posted", d)
}

func (h *CourseHandler) AddReply(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid discussion ID", "VALIDATION_ERROR", nil)
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	rp, err := h.courseUsecase.AddReply(c.Request.Context(), id, viewerFrom(c), req.Content)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Reply posted", rp)
}

func (h *CourseHandler) ToggleUpvote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid discussion ID", "VALIDATION_ERROR", nil)
		return
	}
	n, err := h.courseUsecase.ToggleUpvote(c.Request.Context(), id, viewerFrom(c).UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", gin.H{"upvotes": n})
}

func (h *CourseHandler) GetEnrolledCourses(c *gin.Context) {
	v := viewerFrom(c)
	list, err := h.courseUsecase.GetEnrolledCourses(c.Request.Context(), v.UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *CourseHandler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appErrors.ErrCourseNotFound), errors.Is(err, appErrors.ErrModuleNotFound), errors.Is(err, appErrors.ErrLessonNotFound), errors.Is(err, appErrors.ErrReviewNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, appErrors.ErrForbidden):
		response.Forbidden(c, err.Error())
	case errors.Is(err, appErrors.ErrNotEnrolled):
		response.Unprocessable(c, err.Error())
	default:
		response.Internal(c, err)
	}
}
