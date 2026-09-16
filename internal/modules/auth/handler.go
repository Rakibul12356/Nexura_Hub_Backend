package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
)

type AuthHandler struct {
	authUsecase AuthUsecase
}

func NewAuthHandler(authUsecase AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var dto RegisterDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.authUsecase.Register(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserAlreadyExists) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, appErrors.ErrInvalidRole) {
			response.Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
		return
	}
	response.Success(c, http.StatusCreated, "Registered successfully", res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var dto LoginDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.authUsecase.Login(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, appErrors.ErrInvalidCredentials) {
			response.Unauthorized(c, err.Error())
			return
		}
		if errors.Is(err, appErrors.ErrUserSuspended) {
			response.Forbidden(c, err.Error())
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Logged in successfully", res)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto RefreshTokenDTO
	_ = c.ShouldBindJSON(&dto)
	_ = h.authUsecase.Logout(c.Request.Context(), userID, dto.RefreshToken)
	response.Success(c, http.StatusOK, "Logged out successfully", gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	user, err := h.authUsecase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", user)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var dto UpdateProfileDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	user, err := h.authUsecase.UpdateProfile(c.Request.Context(), userID, dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profile updated successfully", user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var dto ChangePasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.authUsecase.ChangePassword(c.Request.Context(), userID, dto); err != nil {
		if errors.Is(err, appErrors.ErrInvalidCredentials) {
			response.Error(c, http.StatusBadRequest, "Current password is incorrect", "VALIDATION_ERROR", nil)
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Password changed successfully", gin.H{"message": "Password changed successfully"})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var dto RefreshTokenDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.authUsecase.RefreshToken(c.Request.Context(), dto.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "Invalid refresh token")
		return
	}
	response.Success(c, http.StatusOK, "OK", gin.H{
		"token":        res.Token,
		"refreshToken": res.RefreshToken,
		"user":         res.User,
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var dto ForgotPasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	token, err := h.authUsecase.ForgotPassword(c.Request.Context(), dto.Email)
	if err != nil {
		response.Internal(c, err)
		return
	}
	data := gin.H{"message": "Password reset instructions sent to email"}
	if token != "" && gin.Mode() != gin.ReleaseMode {
		data["resetToken"] = token
	}
	response.Success(c, http.StatusOK, "Password reset instructions sent to email", data)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var dto ResetPasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.authUsecase.ResetPassword(c.Request.Context(), dto); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
		return
	}
	response.Success(c, http.StatusOK, "Password reset successfully", gin.H{"message": "Password reset successfully"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var dto VerifyEmailDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.authUsecase.VerifyEmail(c.Request.Context(), dto.Token); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
		return
	}
	response.Success(c, http.StatusOK, "Email verified", gin.H{"message": "Email verified"})
}

func (h *AuthHandler) GetInstructorPublic(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid instructor ID", "VALIDATION_ERROR", nil)
		return
	}
	p, err := h.authUsecase.GetInstructorPublic(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Instructor not found")
		return
	}
	response.Success(c, http.StatusOK, "OK", p)
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, error) {
	return uuid.Parse(c.Param(name))
}
