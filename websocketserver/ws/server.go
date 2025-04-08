package ws

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"websocketserver/auth"
	"websocketserver/models"
)

// upgrader upgrades HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checks.
		return true
	},
}

// Server represents the WebSocket server.
type Server struct {
	db          *sql.DB
	authService *auth.Service
	clients     map[string]*Client // mapping from user_id to client connection
	RateLimiter *RateLimiter       // rate limiter for message processing
	mu          sync.RWMutex
}

// NewServer creates a new WebSocket server instance.
func NewServer(db *sql.DB, authService *auth.Service, messageRate float64, messageBurst int) *Server {
	return &Server{
		db:          db,
		authService: authService,
		clients:     make(map[string]*Client),
		RateLimiter: NewRateLimiter(messageRate, messageBurst),
	}
}

// Client represents an individual WebSocket connection.
type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	server *Server

	// Context for managing goroutine lifecycles.
	ctx    context.Context
	cancel context.CancelFunc
}

// HandleWebSocket upgrades the connection, validates the JWT token, and starts the client's read and write pumps.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate JWT token from the query parameters.
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseToken(tokenStr, s.authService)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	// Upgrade the connection to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create a cancelable context for the client.
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
		ctx:    ctx,
		cancel: cancel,
	}
	s.registerClient(client)

	// Launch the read and write pumps as separate goroutines.
	go client.writePump()
	go client.readPump()
}

// registerClient adds a new client to the server and retrieves any undelivered messages.
func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	s.clients[client.userID] = client
	s.mu.Unlock()
	log.Printf("User %s connected", client.userID)
	// Deliver undelivered messages for this user.
	s.RetrieveUndeliveredMessages(client.userID)
}

// unregisterClient removes a client from the server.
func (s *Server) unregisterClient(client *Client) {
	s.mu.Lock()
	if _, ok := s.clients[client.userID]; ok {
		delete(s.clients, client.userID)
		close(client.send)
	}
	s.mu.Unlock()
	// Clean up rate limiter for this user
	s.RateLimiter.RemoveUser(client.userID)
	log.Printf("User %s disconnected", client.userID)
}

// deliverMessage sends the message to its intended recipient(s).
// For broadcast messages, it iterates over all connected clients (skipping the sender).
func (s *Server) deliverMessage(msg models.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if msg.IsBroadcast {
		s.mu.RLock()
		for id, client := range s.clients {
			// Optionally skip sending back to the sender.
			if id == msg.From {
				continue
			}
			// Non-blocking channel send.
			select {
			case client.send <- data:
			default:
				log.Printf("Warning: send channel for client %s is full", client.userID)
			}
		}
		s.mu.RUnlock()
	} else {
		log.Printf("Attempting to deliver direct message in real time")
		s.mu.RLock()
		recipient, online := s.clients[msg.To]
		s.mu.RUnlock()

		if online {
			select {
			case recipient.send <- data:
				// Update direct message status to "delivered".
				updateQuery := "UPDATE messages SET status = ? WHERE id = ?"
				if _, err := s.db.Exec(updateQuery, "delivered", msg.ID); err != nil {
					log.Printf("Failed to update message status for msg %d: %v", msg.ID, err)
				}
			default:
				log.Printf("Warning: send channel for client %s is full", recipient.userID)
			}
		} else {
			log.Printf("User %s is offline; message %d remains pending", msg.To, msg.ID)
		}
	}
	return nil
}

// readPump continuously reads messages from the WebSocket.
// It uses the client's context for cancellation and ensures a clean shutdown.
func (c *Client) readPump() {
	defer func() {
		c.server.unregisterClient(c)
		c.conn.Close()
		c.cancel()
	}()
	// c.conn.SetReadLimit(512)
	c.conn.SetReadLimit(1024 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("Read error from %s: %v", c.userID, err)
				return
			}
			log.Printf("Received message from %s: %s", c.userID, message)

			// Apply rate limiting
			if !c.server.RateLimiter.Allow(c.userID) {
				log.Printf("Rate limit exceeded for user %s", c.userID)

				// Send rate limit error message to client
				rateLimitErr := models.Message{
					From:    "system",
					To:      c.userID,
					Content: "Rate limit exceeded. Please slow down.",
					Status:  "error",
				}
				if errData, err := json.Marshal(rateLimitErr); err == nil {
					c.send <- errData
				}
				continue
			}

			var msg models.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Invalid message format from %s: %v", c.userID, err)
				continue
			}

			// Determine if the message is a broadcast.
			if msg.To == "broadcast" {
				msg.IsBroadcast = true
			}

			// Save the message with a "pending" status, including the signature if present.
			insertQuery := `INSERT INTO messages (from_user, to_user, timestamp, content, status, is_broadcast, signature) VALUES (?, ?, ?,?, ?, ?, ?)`
			res, err := c.server.db.Exec(insertQuery, msg.From, msg.To, msg.Timestamp, msg.Content, "pending", msg.IsBroadcast, msg.Signature)
			if err != nil {
				log.Printf("Failed to insert message from %s: %v", c.userID, err)
				continue
			}
			lastID, err := res.LastInsertId()
			if err == nil {
				msg.ID = int(lastID)
			}
			// Attempt to deliver the message in real time.
			if err := c.server.deliverMessage(msg); err != nil {
				log.Printf("Delivery error for message %d from %s: %v", msg.ID, c.userID, err)
			}
		}
	}
}

// writePump writes messages from the send channel to the WebSocket.
// It periodically sends pings to keep the connection alive and listens for context cancellation.
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		c.cancel()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed, send a close message.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error to %s: %v", c.userID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error to %s: %v", c.userID, err)
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// RetrieveUndeliveredMessages queries the database for messages that have not yet been delivered
// and attempts to deliver them to the given user.
// It retrieves both direct messages (to_user equals userID) and broadcast messages (is_broadcast = TRUE)
// that have not yet been delivered (as tracked in broadcast_deliveries).
func (s *Server) RetrieveUndeliveredMessages(userID string) {
	query := "SELECT m.id, m.from_user, m.to_user, m.timestamp, m.content, m.status, m.is_broadcast, m.signature FROM messages m LEFT JOIN broadcast_deliveries bd ON m.id = bd.message_id AND bd.user_id = ? WHERE ((m.to_user = ? AND m.status = 'pending') OR (m.is_broadcast = TRUE AND m.status = 'pending')) AND bd.message_id IS NULL"
	rows, err := s.db.Query(query, userID, userID)
	if err != nil {
		log.Printf("Failed to retrieve undelivered messages for %s: %v", userID, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.From, &msg.To, &msg.Timestamp, &msg.Content, &msg.Status, &msg.IsBroadcast, &msg.Signature); err != nil {
			log.Printf("Error scanning message for %s: %v", userID, err)
			continue
		}

		if err := s.deliverMessage(msg); err != nil {
			log.Printf("Error delivering undelivered message %d to %s: %v", msg.ID, userID, err)
		}
		if msg.IsBroadcast {
			insertDeliveryQuery := "INSERT INTO broadcast_deliveries (message_id, user_id) VALUES (?, ?)"
			if _, err := s.db.Exec(insertDeliveryQuery, msg.ID, userID); err != nil {
				log.Printf("Failed to record broadcast delivery for message %d to user %s: %v", msg.ID, userID, err)
			}
		} else {
			updateQuery := "UPDATE messages SET status = ? WHERE id = ?"
			if _, err := s.db.Exec(updateQuery, "delivered", msg.ID); err != nil {
				log.Printf("Failed to update message status for msg %d: %v", msg.ID, err)
			}
		}
	}
}

// exponentialBackoff is an example function to simulate reconnection backoff with jitter.
func exponentialBackoff(base, max time.Duration) time.Duration {
	jitter := time.Duration(float64(base) * (0.5 + 0.5*rand.Float64()))
	if jitter > max {
		return max
	}
	return jitter
}
