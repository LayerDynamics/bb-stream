package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ryanoboyle/bb-stream/pkg/logging"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan Event
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case <-h.done:
			// Close all client connections
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully shuts down the hub
func (h *WebSocketHub) Stop() {
	close(h.done)
}

// Broadcast sends an event to all connected clients
func (h *WebSocketHub) Broadcast(event Event) {
	event.Timestamp = time.Now()
	select {
	case h.broadcast <- event:
	default:
		logging.Logger().Warn("WebSocket broadcast channel full, dropping event",
			"event_type", event.Type)
	}
}

// ClientCount returns the number of connected clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Logger().Error("WebSocket upgrade error", logging.Err(err))
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan Event, 256),
	}

	s.hub.register <- client

	// Start goroutines for reading and writing with panic recovery
	safeGo(func() { client.writePump() })
	safeGo(func() { client.readPump() })

	// Send welcome message
	client.send <- Event{
		Type:      "connected",
		Timestamp: time.Now(),
	}
}

// readPump handles reading messages from the WebSocket
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Logger().Warn("WebSocket read error", logging.Err(err))
			}
			break
		}

		// Handle incoming messages (e.g., subscribe to specific events)
		var msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if err := json.Unmarshal(message, &msg); err == nil {
			// Handle different message types
			switch msg.Type {
			case "ping":
				c.send <- Event{Type: "pong"}
			case "subscribe":
				// Could implement topic-based subscriptions here
			}
		}
	}
}

// writePump handles writing messages to the WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(event); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Progress event types

// UploadProgressEvent represents upload progress
type UploadProgressEvent struct {
	File     string  `json:"file"`
	Percent  float64 `json:"percent"`
	Bytes    int64   `json:"bytes"`
	Total    int64   `json:"total"`
}

// DownloadProgressEvent represents download progress
type DownloadProgressEvent struct {
	File     string  `json:"file"`
	Percent  float64 `json:"percent"`
	Bytes    int64   `json:"bytes"`
	Total    int64   `json:"total"`
}

// SyncProgressEvent represents sync progress
type SyncProgressEvent struct {
	JobID   string `json:"job_id"`
	Phase   string `json:"phase"`
	File    string `json:"file"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
}

// WatchEvent represents a file watch event
type WatchEvent struct {
	JobID  string `json:"job_id"`
	Action string `json:"action"`
	File   string `json:"file"`
	Error  string `json:"error,omitempty"`
}
