package middleware

import (
	"github.com/gin-gonic/gin"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/response"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := CurrentRole(c)
		if userRole == "" {
			response.Unauthorized(c, "Unauthorized")
			c.Abort()
			return
		}
		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}
		response.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

func BlockAdminChatMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentRole(c) == "admin" {
			response.Forbidden(c, appErrors.ErrAdminChatBlocked.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}
