package server

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"nexura-backend/pkg/jwt"
)

func TestSetupServerDoesNotPanicOnRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	jwtSvc := jwt.NewJWTServiceWithRefresh("test-access", "test-refresh", time.Hour, 24*time.Hour)
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("SetupServer panicked: %v", rec)
		}
	}()
	SetupServer(ServerConfig{
		Engine:     r,
		JWTService: jwtSvc,
	})
}
