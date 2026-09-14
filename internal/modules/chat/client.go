// internal/modules/chat/client.go
package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type Client struct {
	Hub         *Hub
	Conn        *websocket.Conn
	Send        chan []byte
	UserID      uuid.UUID
	UserEmail   string
	UserRole    string
	ChatUsecase ChatUsecase
	send        chan []byte
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, email, role string, chatUsecase ChatUsecase) *Client {
	return &Client{
		Hub:         hub,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		send:        make(chan []byte, 256),
		UserID:      userID,
		UserEmail:   email,
		UserRole:    role,
		ChatUsecase: chatUsecase,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS Read error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, []byte("\n"), []byte(" "), -1))
		c.handleIncomingEvent(message)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleIncomingEvent(raw []byte) {
	var event WSEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}

	ctx := context.Background()

	switch event.Event {
	case "join_room":
		var req struct {
			ConversationID string `json:"conversationId"`
		}
		if err := json.Unmarshal(event.Data, &req); err == nil {
			c.Hub.JoinRoom(req.ConversationID, c)
		}

	case "send_message":
		var req struct {
			ConversationID string `json:"conversationId"`
			Message        struct {
				Content  *string `json:"content"`
				ImageURL *string `json:"imageUrl"`
			} `json:"message"`
		}
		if err := json.Unmarshal(event.Data, &req); err == nil {
			convID, err := uuid.Parse(req.ConversationID)
			if err != nil {
				return
			}
			msg, err := c.ChatUsecase.SendMessage(ctx, c.UserID, SendMessageDTO{
				ConversationID: convID,
				Content:        req.Message.Content,
				ImageURL:       req.Message.ImageURL,
			})
			if err == nil {
				broadcastPayload, _ := json.Marshal(map[string]interface{}{
					"event": "receive_message",
					"data": map[string]interface{}{
						"conversationId": req.ConversationID,
						"message":        msg,
					},
				})
				c.Hub.BroadcastToRoom(req.ConversationID, broadcastPayload)
			}
		}

	case "send_reaction":
		var req struct {
			ConversationID string `json:"conversationId"`
			MessageID      string `json:"messageId"`
			Emoji          string `json:"emoji"`
			UserID         string `json:"userId"`
		}
		if err := json.Unmarshal(event.Data, &req); err == nil {
			msgID, err := uuid.Parse(req.MessageID)
			if err != nil {
				return
			}
			convID, err := uuid.Parse(req.ConversationID)
			if err != nil {
				return
			}
			_, _ = c.ChatUsecase.ToggleReaction(ctx, c.UserID, SendReactionDTO{
				ConversationID: convID,
				MessageID:      msgID,
				Emoji:          req.Emoji,
			})

			broadcastPayload, _ := json.Marshal(map[string]interface{}{
				"event": "receive_reaction",
				"data": map[string]interface{}{
					"conversationId": req.ConversationID,
					"messageId":      req.MessageID,
					"emoji":          req.Emoji,
					"userId":         c.UserID.String(),
				},
			})
			c.Hub.BroadcastToRoom(req.ConversationID, broadcastPayload)
		}

	case "typing_status":
		var req struct {
			ConversationID string `json:"conversationId"`
			UserID         string `json:"userId"`
			UserName       string `json:"userName"`
			IsTyping       bool   `json:"isTyping"`
		}
		if err := json.Unmarshal(event.Data, &req); err == nil {
			broadcastPayload, _ := json.Marshal(map[string]interface{}{
				"event": "typing_status",
				"data":  req,
			})
			c.Hub.BroadcastToRoom(req.ConversationID, broadcastPayload)
		}
	}
}
