package chat

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/jwt"
)

func NewSocketServer(jwtService *jwt.JWTService, chatUsecase ChatUsecase) *socketio.Server {
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		token := bearerFromHeader(s.RemoteHeader().Get("Authorization"))
		u := s.URL()
		if token == "" {
			token = u.Query().Get("token")
		}
		userID := uuid.Nil
		role := ""
		if token != "" {
			if claims, err := jwtService.ValidateToken(token); err == nil {
				userID = claims.UserID
				role = claims.Role
			}
		}
		if qid := u.Query().Get("userId"); userID == uuid.Nil && qid != "" {
			userID, _ = uuid.Parse(qid)
			role = u.Query().Get("userRole")
		}
		if role == "admin" {
			s.Emit("error", appErrors.ErrAdminChatBlocked.Error())
			s.Close()
			return appErrors.ErrAdminChatBlocked
		}
		s.SetContext(socketAuth{UserID: userID, Role: role})
		if userID != uuid.Nil {
			s.Join("user:" + userID.String())
			server.BroadcastToNamespace("/", "user_presence", map[string]any{"userId": userID.String(), "isOnline": true})
		}
		return nil
	})

	server.OnEvent("/", "join_room", func(s socketio.Conn, data map[string]any) {
		if id, _ := data["conversationId"].(string); id != "" {
			s.Join(id)
		}
	})
	server.OnEvent("/", "leave_room", func(s socketio.Conn, data map[string]any) {
		if id, _ := data["conversationId"].(string); id != "" {
			s.Leave(id)
		}
	})
	server.OnEvent("/", "send_message", func(s socketio.Conn, data map[string]any) {
		auth, _ := s.Context().(socketAuth)
		if auth.UserID == uuid.Nil {
			return
		}
		convStr, _ := data["conversationId"].(string)
		convID, err := uuid.Parse(convStr)
		if err != nil {
			return
		}
		content := ""
		var imageURL *string
		if msg, ok := data["message"].(map[string]any); ok {
			if c, ok := msg["content"].(string); ok {
				content = c
			}
			if u, ok := msg["imageUrl"].(string); ok && u != "" {
				imageURL = &u
			}
		}
		saved, err := chatUsecase.SendMessage(context.Background(), auth.UserID, SendMessageDTO{
			ConversationID: convID, Content: content, ImageURL: imageURL,
		})
		if err != nil {
			s.Emit("error", err.Error())
			return
		}
		payload := map[string]any{"conversationId": convStr, "message": saved}
		server.BroadcastToRoom("/", convStr, "receive_message", payload)
	})
	server.OnEvent("/", "typing_status", func(s socketio.Conn, data map[string]any) {
		if id, _ := data["conversationId"].(string); id != "" {
			server.BroadcastToRoom("/", id, "user_typing", data)
		}
	})
	server.OnEvent("/", "send_reaction", func(s socketio.Conn, data map[string]any) {
		auth, _ := s.Context().(socketAuth)
		convStr, _ := data["conversationId"].(string)
		msgStr, _ := data["messageId"].(string)
		emoji, _ := data["emoji"].(string)
		msgID, err := uuid.Parse(msgStr)
		if err == nil && auth.UserID != uuid.Nil {
			convID, _ := uuid.Parse(convStr)
			_, _ = chatUsecase.ToggleReaction(context.Background(), auth.UserID, SendReactionDTO{
				ConversationID: convID, MessageID: msgID, Emoji: emoji,
			})
		}
		if convStr != "" {
			data["userId"] = auth.UserID.String()
			server.BroadcastToRoom("/", convStr, "receive_reaction", data)
		}
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		auth, _ := s.Context().(socketAuth)
		if auth.UserID != uuid.Nil {
			server.BroadcastToNamespace("/", "user_presence", map[string]any{"userId": auth.UserID.String(), "isOnline": false})
		}
		log.Printf("[Socket.IO] disconnect: %s", reason)
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("[Socket.IO] error: %v", e)
	})
	return server
}

type socketAuth struct {
	UserID uuid.UUID
	Role   string
}

func bearerFromHeader(h string) string {
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
