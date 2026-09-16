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
	if t := fromAuthHeader(c.GetHeader("Authorization")); t != "" {
		return t
	}
	if q := strings.TrimSpace(c.Query("token")); q != "" {
		return q
	}
	for _, name := range []string{"nexurahub_token", "nexurahub_refresh_token"} {
		if ck, err := c.Cookie(name); err == nil {
			if t := strings.TrimSpace(ck); t != "" {
				return t
			}
		}
	}
	return ""
}

func fromAuthHeader(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	if strings.Count(h, ".") == 2 {
		return h
	}
	return ""
}

func setClaims(c *gin.Context, jwtService *jwt.JWTService, token string) bool {
	if jwtService == nil {
		return false
	}
	claims, err := jwtService.ValidateAccessOrRefresh(token)
	if err != nil || claims == nil || claims.UserID == uuid.Nil {
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
	switch id := v.(type) {
	case uuid.UUID:
		return id, id != uuid.Nil
	case string:
		parsed, err := uuid.Parse(strings.TrimSpace(id))
		return parsed, err == nil && parsed != uuid.Nil
	default:
		return uuid.Nil, false
	}
}

func CurrentRole(c *gin.Context) string {
	v, _ := c.Get("role")
	role, _ := v.(string)
	return strings.ToLower(strings.TrimSpace(role))
}
