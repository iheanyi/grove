package dashboard

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	send   chan Message
	hub    *Hub
	topics map[string]bool
	mu     sync.Mutex
}

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client send buffer is full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Use the websocket package's handler
	websocket.Handler(func(ws *websocket.Conn) {
		client := &Client{
			conn:   ws,
			send:   make(chan Message, 256),
			hub:    h,
			topics: make(map[string]bool),
		}

		h.register <- client

		// Start writer goroutine
		go client.writePump()

		// Read messages (mainly for subscription management)
		client.readPump()

		h.unregister <- client
	}).ServeHTTP(w, r)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer c.conn.Close()

	for {
		var msg Message
		err := websocket.JSON.Receive(c.conn, &msg)
		if err != nil {
			break
		}

		// Handle subscription messages
		if msg.Type == "subscribe" {
			if topics, ok := msg.Payload.([]interface{}); ok {
				c.mu.Lock()
				for _, t := range topics {
					if topic, ok := t.(string); ok {
						c.topics[topic] = true
					}
				}
				c.mu.Unlock()
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Channel closed
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				continue
			}

			if _, err := c.conn.Write(data); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			ping := Message{Type: "ping"}
			data, _ := json.Marshal(ping)
			if _, err := c.conn.Write(data); err != nil {
				return
			}
		}
	}
}
