package payment

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) DummyPay(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto DummyPayDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.repo.DummyPay(c.Request.Context(), userID, dto.CourseID, dto.CouponCode, dto.Gateway)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Dummy payment completed. Course enrolled.", res)
}

func (h *Handler) FreeEnroll(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto FreeEnrollDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.repo.FreeEnroll(c.Request.Context(), userID, dto.CourseID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Enrolled successfully", res)
}

func (h *Handler) EnrollByCourseParam(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	res, err := h.repo.FreeEnroll(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, appErrors.ErrFreeEnrollOnly) {
			res, err = h.repo.DummyPay(c.Request.Context(), userID, c.Param("id"), "", "dummy")
		}
	}
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Enrolled successfully", res)
}

func (h *Handler) GetPayment(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	res, err := h.repo.GetPayment(c.Request.Context(), c.Param("id"), userID, middleware.CurrentRole(c) == "admin")
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", res)
}

func (h *Handler) ListCoupons(c *gin.Context) {
	list, err := h.repo.ListCoupons(c.Request.Context())
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto CouponDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	cp := &Coupon{Code: dto.Code, DiscountType: dto.DiscountType, DiscountValue: dto.DiscountValue, ExpiryDate: dto.ExpiryDate, MaxRedemptions: dto.MaxRedemptions}
	if err := h.repo.CreateCoupon(c.Request.Context(), cp, userID); err != nil {
		if isDup(err) {
			response.Conflict(c, "Coupon code already exists")
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Coupon created", cp)
}

func (h *Handler) PatchCoupon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var dto struct {
		IsActive *bool `json:"isActive"`
	}
	_ = c.ShouldBindJSON(&dto)
	cp, err := h.repo.PatchCoupon(c.Request.Context(), id, dto.IsActive)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Coupon updated", cp)
}

func (h *Handler) DeleteCoupon(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.repo.DeleteCoupon(c.Request.Context(), id); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Coupon deleted", gin.H{})
}

func (h *Handler) ValidateCoupon(c *gin.Context) {
	var dto ValidateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.repo.ValidateCoupon(c.Request.Context(), dto.Code, dto.CourseID)
	if err != nil {
		response.Unprocessable(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "OK", res)
}

func (h *Handler) AdminWallet(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	w, err := h.repo.GetWallet(c.Request.Context(), "admin", userID)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", w)
}

func (h *Handler) InstructorWallet(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	w, err := h.repo.GetWallet(c.Request.Context(), "instructor", userID)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", w)
}

func (h *Handler) EnrollmentStatus(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	ref := c.Param("id")
	if ref == "" {
		ref = c.Param("courseId")
	}
	res, err := h.repo.EnrollmentStatus(c.Request.Context(), userID, ref)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", res)
}

func (h *Handler) CourseEnrollments(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		courseID, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	list, err := h.repo.ListCourseEnrollments(c.Request.Context(), courseID, c.Query("search"))
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *Handler) CompleteLesson(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("lessonId"))
	if err != nil {
		id, err = uuid.Parse(c.Param("id"))
	}
	if err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.repo.CompleteLesson(c.Request.Context(), userID, id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Lesson marked as completed", res)
}

func isDup(err error) bool {
	return err != nil && (errors.Is(err, appErrors.ErrUserAlreadyExists) || containsUnique(err.Error()))
}

func containsUnique(s string) bool {
	return len(s) > 0 && (contains(s, "duplicate") || contains(s, "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appErrors.ErrAlreadyEnrolled):
		response.Conflict(c, err.Error())
	case errors.Is(err, appErrors.ErrCourseNotFound), errors.Is(err, appErrors.ErrPaymentNotFound), errors.Is(err, appErrors.ErrLessonNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, appErrors.ErrCouponInvalid), errors.Is(err, appErrors.ErrNotEnrolled), errors.Is(err, appErrors.ErrFreeEnrollOnly):
		response.Unprocessable(c, err.Error())
	case errors.Is(err, appErrors.ErrCourseNotPublished):
		response.Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
	case errors.Is(err, appErrors.ErrForbidden):
		response.Forbidden(c, err.Error())
	default:
		response.Internal(c, err)
	}
}
