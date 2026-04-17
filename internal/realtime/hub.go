package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Event struct {
	Event   string `json:"event"`
	Room    string `json:"room,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

type Hub struct {
	redis       *redis.Client
	useRedis    bool
	mu          sync.RWMutex
	clients     map[uint64]map[*websocket.Conn]bool
	roomClients map[string]map[*websocket.Conn]bool
}

func NewHub(redis *redis.Client, useRedis bool) *Hub {
	return &Hub{
		redis:       redis,
		useRedis:    useRedis,
		clients:     map[uint64]map[*websocket.Conn]bool{},
		roomClients: map[string]map[*websocket.Conn]bool{},
	}
}

func (h *Hub) Publish(ctx context.Context, room string, event Event) {
	event.Room = room
	data, _ := json.Marshal(event)
	if h.useRedis && h.redis != nil {
		_ = h.redis.Publish(ctx, "room:"+room, data).Err()
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.roomClients[room] {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *Hub) PushUser(userID uint64, event Event) {
	data, _ := json.Marshal(event)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients[userID] {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, userID uint64, rooms []string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.register(userID, conn, rooms)
	defer func() {
		h.unregister(userID, conn, rooms)
		_ = conn.Close()
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	if h.useRedis && h.redis != nil {
		pubsub := h.redis.Subscribe(ctx, channelNames(rooms)...)
		defer pubsub.Close()

		go func() {
			for msg := range pubsub.Channel() {
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					cancel()
					return
				}
			}
		}()
	}

	for {
		_, body, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var incoming Event
		if err := json.Unmarshal(body, &incoming); err != nil {
			log.Printf("invalid websocket event: %v", err)
			continue
		}
		if incoming.Event == "USER_TYPING" && incoming.Room != "" {
			h.Publish(ctx, incoming.Room, incoming)
		}
	}
}

func (h *Hub) register(userID uint64, conn *websocket.Conn, rooms []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*websocket.Conn]bool{}
	}
	h.clients[userID][conn] = true
	if !h.useRedis {
		for _, room := range rooms {
			if h.roomClients[room] == nil {
				h.roomClients[room] = map[*websocket.Conn]bool{}
			}
			h.roomClients[room][conn] = true
		}
	}
}

func (h *Hub) unregister(userID uint64, conn *websocket.Conn, rooms []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[userID], conn)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
	for _, room := range rooms {
		delete(h.roomClients[room], conn)
		if len(h.roomClients[room]) == 0 {
			delete(h.roomClients, room)
		}
	}
}

func channelNames(rooms []string) []string {
	if len(rooms) == 0 {
		return []string{"room:global"}
	}
	out := make([]string, 0, len(rooms))
	for _, room := range rooms {
		out = append(out, "room:"+room)
	}
	return out
}
