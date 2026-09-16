package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultOrigins = []string{
	"https://nexurahub.vercel.app",
	"https://www.nexurahub.vercel.app",
	"http://localhost:5173",
	"http://localhost:3000",
	"http://127.0.0.1:5173",
	"http://127.0.0.1:3000",
}

func CORSMiddleware() gin.HandlerFunc {
	return CORSWithOrigin("*")
}

func CORSWithOrigin(origin string) gin.HandlerFunc {
	allowAny, allowed := parseOrigins(origin)
	return func(c *gin.Context) {
		reqOrigin := strings.TrimSpace(c.GetHeader("Origin"))
		allow := matchOrigin(reqOrigin, allowAny, allowed)
		if allow != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allow)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseOrigins(raw string) (allowAny bool, allowed map[string]struct{}) {
	allowed = make(map[string]struct{}, len(defaultOrigins)+4)
	for _, o := range defaultOrigins {
		allowed[strings.ToLower(o)] = struct{}{}
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return true, allowed
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(strings.TrimSuffix(part, "/"))
		if part == "*" {
			return true, allowed
		}
		if part != "" {
			allowed[strings.ToLower(part)] = struct{}{}
		}
	}
	return false, allowed
}

func matchOrigin(reqOrigin string, allowAny bool, allowed map[string]struct{}) string {
	reqOrigin = strings.TrimSpace(strings.TrimSuffix(reqOrigin, "/"))
	if reqOrigin == "" {
		if allowAny {
			return "*"
		}
		return ""
	}
	if _, ok := allowed[strings.ToLower(reqOrigin)]; ok {
		return reqOrigin
	}
	if allowAny {
		return reqOrigin
	}
	return ""
}
