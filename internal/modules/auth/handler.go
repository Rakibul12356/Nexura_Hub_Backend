// internal/modules/auth/handler.go
package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
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
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	res, err := h.authUsecase.Register(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   res,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var dto LoginDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	res, err := h.authUsecase.Login(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, appErrors.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": err.Error()})
			return
		}
		if errors.Is(err, appErrors.ErrUserSuspended) {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   res,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	_ = h.authUsecase.Logout(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "success",
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}

	userID := userIDVal.(uuid.UUID)
	user, err := h.authUsecase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var dto UpdateProfileDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	user, err := h.authUsecase.UpdateProfile(c.Request.Context(), userID, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile updated successfully",
		"data":    user,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
		return
	}
	userID := userIDVal.(uuid.UUID)

	var dto ChangePasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if err := h.authUsecase.ChangePassword(c.Request.Context(), userID, dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password changed successfully",
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var dto RefreshTokenDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	res, err := h.authUsecase.RefreshToken(c.Request.Context(), dto.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   res,
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var dto ForgotPasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if err := h.authUsecase.ForgotPassword(c.Request.Context(), dto.Email); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password reset instructions sent to email",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var dto ResetPasswordDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	if err := h.authUsecase.ResetPassword(c.Request.Context(), dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Password reset successfully",
	})
}
