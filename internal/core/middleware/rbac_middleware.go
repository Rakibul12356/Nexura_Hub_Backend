// internal/core/middleware/rbac_middleware.go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appErrors "nexura-backend/internal/core/errors"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
			c.Abort()
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Invalid user role"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Insufficient permissions"})
		c.Abort()
	}
}

func BlockAdminChatMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if exists {
			if userRole, ok := roleVal.(string); ok && userRole == "admin" {
				c.JSON(http.StatusForbidden, gin.H{
					"status":  "error",
					"message": appErrors.ErrAdminChatBlocked.Error(),
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
