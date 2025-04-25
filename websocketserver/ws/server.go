package ws

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"websocketserver/auth"
	"websocketserver/metrics"
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
	db               *sql.DB
	authService      *auth.Service
	clients          map[string]*Client // mapping from user_id to client connection
	RateLimiter      *RateLimiter       // rate limiter for message processing
	mu               sync.RWMutex
	responseChannels map[string]chan models.Message // mapping from user_id to response channels
	responseMu       sync.RWMutex                   // mutex for response channels
}

// NewServer creates a new WebSocket server instance.
func NewServer(db *sql.DB, authService *auth.Service, messageRate float64, messageBurst int) *Server {
	return &Server{
		db:               db,
		authService:      authService,
		clients:          make(map[string]*Client),
		RateLimiter:      NewRateLimiter(messageRate, messageBurst),
		responseChannels: make(map[string]chan models.Message),
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
		http.Error(w, "Missing authentication token", http.StatusUnauthorized)
		return
	}

	// Use enhanced token verification
	tokenResult := auth.VerifyToken(tokenStr, s.authService, "")
	if !tokenResult.Valid || tokenResult.Error != nil {
		http.Error(w, fmt.Sprintf("Invalid token: %v", tokenResult.Error), http.StatusUnauthorized)
		return
	}

	// Get user ID from verified token
	userID := tokenResult.UserID

	// Verify that the user exists in the database
	var userExists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?)", userID).Scan(&userExists)
	if err != nil {
		log.Printf("Database error checking user existence: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !userExists {
		log.Printf("Security alert: Token contains valid signature but non-existent user ID: %s", userID)
		http.Error(w, "Invalid user", http.StatusUnauthorized)
		return
	}

	// Log connection for security auditing
	log.Printf("Authenticated WebSocket connection for user %s", userID)

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

// UserStatusResponse represents the JSON response for user connectivity.
type UserStatusResponse struct {
	Online  []string `json:"online"`
	Offline []string `json:"offline"`
}

// ActiveUsersHandler returns a JSON struct with "online" and "offline" user lists.
func (s *Server) ActiveUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Only support GET requests.
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve online users from the in-memory clients map.
	s.mu.RLock()
	onlineUsers := make([]string, 0, len(s.clients))
	for userID := range s.clients {
		onlineUsers = append(onlineUsers, userID)
	}
	s.mu.RUnlock()

	// Prepare a set for fast lookup.
	onlineSet := make(map[string]bool, len(onlineUsers))
	for _, id := range onlineUsers {
		onlineSet[id] = true
	}

	// Retrieve all registered users from the database.
	rows, err := s.db.Query("SELECT user_id FROM users")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	allUsers := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		allUsers = append(allUsers, userID)
	}

	// Determine offline users by subtracting online users from all registered users.
	offlineUsers := make([]string, 0)
	for _, id := range allUsers {
		if !onlineSet[id] {
			offlineUsers = append(offlineUsers, id)
		}
	}

	// Compose the response struct.
	resp := UserStatusResponse{
		Online:  onlineUsers,
		Offline: offlineUsers,
	}

	// Write the JSON response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// registerClient adds a new client to the server and retrieves any undelivered messages.
func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	s.clients[client.userID] = client
	s.mu.Unlock()
	log.Printf("User %s connected", client.userID)

	// Create a unique session ID using the client pointer.
	sessionID := fmt.Sprintf("%p", client)
	// Record session start in memory (if desired) and in the database.
	metrics.RecordSessionStart(sessionID, client.userID)
	metrics.RecordSessionStartPersist(sessionID, client.userID, time.Now())

	// Instrumentation: record session start (using the client pointer as a sessionID)
	// sessionID := fmt.Sprintf("%p", client)
	// metrics.RecordSessionStart(sessionID, client.userID)
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
	// Record session end both in memory and persist to the database.
	sessionID := fmt.Sprintf("%p", client)
	metrics.RecordSessionEnd(sessionID, client.userID)
	metrics.RecordSessionEndPersist(sessionID, time.Now())
	// // Instrumentation: record session end.
	// sessionID := fmt.Sprintf("%p", client)
	// metrics.RecordSessionEnd(sessionID, client.userID)
	log.Printf("User %s disconnected", client.userID)
}

// Add this new parameter to deliverMessage
// deliverMessage sends the message to its intended recipient(s).
// For broadcast messages, it iterates over all connected clients (skipping the sender).
// If isReconnection is true, it only delivers to the specified targetUser.
func (s *Server) deliverMessage(msg models.Message, isReconnection bool, targetUser string) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if msg.IsBroadcast {
		s.mu.RLock()

		// If this is a reconnection delivery, only send to the target user
		if isReconnection {
			if client, ok := s.clients[targetUser]; ok {
				select {
				case client.send <- data:
					// Record this broadcast delivery
					insertDeliveryQuery := "INSERT INTO broadcast_deliveries (message_id, user_id) VALUES (?, ?)"
					if _, err := s.db.Exec(insertDeliveryQuery, msg.ID, targetUser); err != nil {
						log.Printf("Failed to record broadcast delivery for message %d to user %s: %v", msg.ID, targetUser, err)
					}
				default:
					log.Printf("Warning: send channel for client %s is full", client.userID)
				}
			}
		} else {
			// Regular broadcast to all connected clients (except sender)
			for id, client := range s.clients {
				// Skip sending back to the sender
				if id == msg.From {
					continue
				}
				// Non-blocking channel send
				select {
				case client.send <- data:
				default:
					log.Printf("Warning: send channel for client %s is full", client.userID)
				}
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
				// Update direct message status to "delivered"
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

// RegisterResponseChannel creates and registers a response channel for a user
func (s *Server) RegisterResponseChannel(userID string) chan models.Message {
	ch := make(chan models.Message, 1) // Buffer size of 1 to prevent blocking

	s.responseMu.Lock()
	s.responseChannels[userID] = ch
	s.responseMu.Unlock()

	return ch
}

// GetResponseChannel retrieves a response channel for a user
func (s *Server) GetResponseChannel(userID string) (chan models.Message, bool) {
	s.responseMu.RLock()
	ch, ok := s.responseChannels[userID]
	s.responseMu.RUnlock()

	return ch, ok
}

// RemoveResponseChannel removes a response channel for a user
func (s *Server) RemoveResponseChannel(userID string) {
	s.responseMu.Lock()
	delete(s.responseChannels, userID)
	s.responseMu.Unlock()
}

// DeliverHTTPMessage delivers a message sent via HTTP to a WebSocket connection
// This is used for the direct message API endpoint
func (s *Server) DeliverHTTPMessage(msg models.Message) error {
	// First, save the message in the database
	insertQuery := `INSERT INTO messages (from_user, to_user, timestamp, content, status, is_broadcast, signature, is_forward_message) 
	                VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := s.db.Exec(insertQuery, msg.From, msg.To, msg.Timestamp, msg.Content,
		"pending", false, msg.Signature, msg.IsForwardMessage)
	if err != nil {
		log.Printf("Failed to insert HTTP message from %s to %s: %v", msg.From, msg.To, err)
		return err
	}

	lastID, err := res.LastInsertId()
	if err == nil {
		msg.ID = int(lastID)
	} else {
		log.Printf("Failed to get last insert ID for HTTP message: %v", err)
	}

	// Now attempt to deliver the message using the existing mechanism
	return s.deliverMessage(msg, false, "")
}

// Modify readPump to use the updated deliverMessage signature
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
			// Instrumentation: record message sent (using client pointer as sessionID).
			sessionID := fmt.Sprintf("%p", c)

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

			// Check if this is a forward response message by either:
			// 1. The message is marked with IsForwardMessage flag
			// 2. The content has "type":"forward_response"
			if msg.IsForwardMessage {
				log.Printf("Received message with IsForwardMessage flag from %s", c.userID)

				// Forward messages should be sent to response channels regardless of content
				c.server.responseMu.RLock()
				responseCh, exists := c.server.responseChannels[c.userID]
				c.server.responseMu.RUnlock()

				if exists {
					// Send the message to the response channel
					select {
					case responseCh <- msg:
						log.Printf("Forward response sent to channel for user %s", c.userID)
					default:
						log.Printf("Warning: response channel for user %s is full or closed", c.userID)
					}
					continue // Skip normal message processing
				}
			} else {
				// Fallback check by looking at message content for legacy clients
				var content map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Content), &content); err == nil {
					// Check if it's a forward_response type
					if typ, ok := content["type"].(string); ok && typ == "forward_response" {
						log.Printf("Received forward response message by content type from %s", c.userID)

						// Check if there's a waiting response channel
						c.server.responseMu.RLock()
						responseCh, exists := c.server.responseChannels[c.userID]
						c.server.responseMu.RUnlock()

						if exists {
							// Send the message to the response channel
							select {
							case responseCh <- msg:
								log.Printf("Forward response sent to channel for user %s", c.userID)
							default:
								log.Printf("Warning: response channel for user %s is full or closed", c.userID)
							}
							continue // Skip normal message processing
						}
					}
				}
			}

			// Use the client pointer as the session ID for persistence.
			metrics.RecordMessageSent(sessionID, msg.IsBroadcast)
			metrics.RecordMessageEventPersist(sessionID, c.userID, msg.IsBroadcast, time.Now())

			// Save the message with a "pending" status, including the signature if present.
			insertQuery := `INSERT INTO messages (from_user, to_user, timestamp, content, status, is_broadcast, signature, is_forward_message) 
                           VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
			res, err := c.server.db.Exec(insertQuery, msg.From, msg.To, msg.Timestamp, msg.Content,
				"pending", msg.IsBroadcast, msg.Signature, msg.IsForwardMessage)
			if err != nil {
				log.Printf("Failed to insert message from %s: %v", c.userID, err)
				continue
			}
			lastID, err := res.LastInsertId()
			if err == nil {
				msg.ID = int(lastID)
			}
			// Attempt to deliver the message in real time.
			// Pass false for isReconnection and empty string for targetUser since this is a normal message delivery
			if err := c.server.deliverMessage(msg, false, ""); err != nil {
				log.Printf("Delivery error for message %d from %s: %v", msg.ID, c.userID, err)
			}
		}
	}
}

// Update the RetrieveUndeliveredMessages function to use the updated deliverMessage
func (s *Server) RetrieveUndeliveredMessages(userID string) {
	// Get the user's registration time
	var createdAt time.Time
	err := s.db.QueryRow("SELECT created_at FROM users WHERE user_id = ?", userID).Scan(&createdAt)
	if err != nil {
		log.Printf("Failed to retrieve user registration time for %s: %v", userID, err)
		// If we can't get the registration time, proceed with caution - just deliver direct messages
		query := `
            SELECT m.id, m.from_user, m.to_user, m.timestamp, m.content, m.status, m.is_broadcast, m.signature 
            FROM messages m 
            LEFT JOIN broadcast_deliveries bd ON m.id = bd.message_id AND bd.user_id = ? 
            WHERE m.to_user = ? AND m.status = 'pending' AND bd.message_id IS NULL
        `
		rows, err := s.db.Query(query, userID, userID)
		if err != nil {
			log.Printf("Failed to retrieve direct messages for %s: %v", userID, err)
			return
		}
		defer rows.Close()

		// Update to use the new version of processMessages
		processMessages(s, rows, userID)
		return
	}

	// Query for undelivered messages, including both direct and broadcast messages
	// For broadcast messages, we rely on the database's automatic timestamp
	query := `
        SELECT m.id, m.from_user, m.to_user, m.timestamp, m.content, m.status, m.is_broadcast, m.signature 
        FROM messages m 
        LEFT JOIN broadcast_deliveries bd ON m.id = bd.message_id AND bd.user_id = ? 
        WHERE (
            (m.to_user = ? AND m.status = 'pending') 
            OR 
            (m.is_broadcast = TRUE AND m.status = 'pending' AND datetime(m.timestamp) >= datetime(?))
        ) 
        AND bd.message_id IS NULL
    `

	rows, err := s.db.Query(query, userID, userID, createdAt)
	if err != nil {
		log.Printf("Failed to retrieve undelivered messages for %s: %v", userID, err)
		return
	}
	defer rows.Close()

	// Update to use the new version of processMessages
	processMessages(s, rows, userID)
}

// Update the helper function to process the message rows
func processMessages(s *Server, rows *sql.Rows, userID string) {
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.From, &msg.To, &msg.Timestamp, &msg.Content, &msg.Status, &msg.IsBroadcast, &msg.Signature); err != nil {
			log.Printf("Error scanning message for %s: %v", userID, err)
			continue
		}

		// Pass true for isReconnection and userID for targetUser since this is a reconnection delivery
		if err := s.deliverMessage(msg, true, userID); err != nil {
			log.Printf("Error delivering undelivered message %d to %s: %v", msg.ID, userID, err)
		}

		// For direct messages, update the status
		if !msg.IsBroadcast {
			updateQuery := "UPDATE messages SET status = ? WHERE id = ?"
			if _, err := s.db.Exec(updateQuery, "delivered", msg.ID); err != nil {
				log.Printf("Failed to update message status for msg %d: %v", msg.ID, err)
			}
		}
		// Note: broadcast_deliveries are now handled in the deliverMessage function
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

// exponentialBackoff is an example function to simulate reconnection backoff with jitter.
func exponentialBackoff(base, max time.Duration) time.Duration {
	jitter := time.Duration(float64(base) * (0.5 + 0.5*rand.Float64()))
	if jitter > max {
		return max
	}
	return jitter
}
