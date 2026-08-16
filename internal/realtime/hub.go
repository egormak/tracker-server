package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type EventType string

const (
	EventTaskStarted  EventType = "TASK_STARTED"
	EventTaskPaused   EventType = "TASK_PAUSED"
	EventTaskResumed  EventType = "TASK_RESUMED"
	EventTaskStopped  EventType = "TASK_STOPPED"
	EventHeartbeatAck EventType = "HEARTBEAT_ACK"
	EventStateSync    EventType = "STATE_SYNC"
)

type Event struct {
	Type       EventType   `json:"type"`
	TaskName   string      `json:"task_name,omitempty"`
	Role       string      `json:"role,omitempty"`
	Duration   int         `json:"duration,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	ServerTime int64       `json:"server_time"` // unix milliseconds
	Data       interface{} `json:"data,omitempty"`
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Event
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan Event, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("Realtime Hub: client connected", "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			slog.Info("Realtime Hub: client disconnected", "total_clients", len(h.clients))

		case event := <-h.broadcast:
			if event.ServerTime == 0 {
				event.ServerTime = time.Now().UnixMilli()
			}
			data, err := json.Marshal(event)
			if err != nil {
				slog.Error("Realtime Hub: failed to marshal event", "error", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					slog.Warn("Realtime Hub: error writing to client, scheduling unregister", "error", err)
					client.Close()
					// We unregister in a separate goroutine or handle on next lock
					go func(c *websocket.Conn) {
						h.Unregister(c)
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

func (h *Hub) Broadcast(event Event) {
	if event.ServerTime == 0 {
		event.ServerTime = time.Now().UnixMilli()
	}
	select {
	case h.broadcast <- event:
	default:
		slog.Warn("Realtime Hub: broadcast channel buffer full, event dropped", "event_type", event.Type)
	}
}
