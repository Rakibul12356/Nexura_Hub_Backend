package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/jwt"
)

func JWTAuthMiddleware(jwtService *jwt.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}
		if !setClaims(c, jwtService, token) {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}
		c.Next()
	}
}

func OptionalJWTMiddleware(jwtService *jwt.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token != "" {
			_ = setClaims(c, jwtService, token)
		}
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		if q := c.Query("token"); q != "" {
			return q
		}
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func setClaims(c *gin.Context, jwtService *jwt.JWTService, token string) bool {
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		return false
	}
	c.Set("userID", claims.UserID)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)
	c.Set("claims", claims)
	return true
}

func CurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("userID")
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func CurrentRole(c *gin.Context) string {
	v, _ := c.Get("role")
	role, _ := v.(string)
	return role
}

