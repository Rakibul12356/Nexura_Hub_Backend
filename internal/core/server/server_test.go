package server

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetupServerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Ensure SetupServer initializes all routes without any route conflict panics
	SetupServer(ServerConfig{
		Engine: engine,
	})

	routes := engine.Routes()
	if len(routes) == 0 {
		t.Fatal("expected registered routes, got 0")
	}
}
