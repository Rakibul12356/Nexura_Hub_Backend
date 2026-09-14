// internal/modules/chat/hub.go
package chat

import (
	"encoding/json"
	"log"
	"sync"
)

type WSEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *RoomMessage
	mutex      sync.RWMutex
}

type RoomMessage struct {
	RoomID  string
	Sender  *Client
	Content []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *RoomMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("[WS Hub] Client registered: user %s", client.UserID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				for roomID, clientsInRoom := range h.rooms {
					if _, joined := clientsInRoom[client]; joined {
						delete(clientsInRoom, client)
						if len(clientsInRoom) == 0 {
							delete(h.rooms, roomID)
						}
					}
				}
			}
			h.mutex.Unlock()
			log.Printf("[WS Hub] Client unregistered: user %s", client.UserID)

		case msg := <-h.broadcast:
			h.mutex.RLock()
			if clientsInRoom, ok := h.rooms[msg.RoomID]; ok {
				for client := range clientsInRoom {
					select {
					case client.send <- msg.Content:
					default:
						close(client.send)
						delete(h.clients, client)
						delete(clientsInRoom, client)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) JoinRoom(roomID string, client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	log.Printf("[WS Hub] User %s joined room %s", client.UserID, roomID)
}

func (h *Hub) BroadcastToRoom(roomID string, content []byte) {
	h.broadcast <- &RoomMessage{
		RoomID:  roomID,
		Content: content,
	}
}
