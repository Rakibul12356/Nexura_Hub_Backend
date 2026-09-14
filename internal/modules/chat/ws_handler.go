// internal/modules/chat/ws_handler.go
package chat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/jwt"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub         *Hub
	jwtService  *jwt.JWTService
	chatUsecase ChatUsecase
}

func NewWSHandler(hub *Hub, jwtService *jwt.JWTService, chatUsecase ChatUsecase) *WSHandler {
	return &WSHandler{
		hub:         hub,
		jwtService:  jwtService,
		chatUsecase: chatUsecase,
	}
}

func (h *WSHandler) ServeWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Token query param required"})
		return
	}

	claims, err := h.jwtService.ValidateToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid token"})
		return
	}

	if claims.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": appErrors.ErrAdminChatBlocked.Error()})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, claims.UserID, claims.Email, claims.Role, h.chatUsecase)
	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
