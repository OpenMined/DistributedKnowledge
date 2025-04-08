package ws

import (
	// "database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/websocket"
	"websocketserver/auth"
	"websocketserver/models"
)

// MockConn implements a mock websocket.Conn for testing
type MockConn struct {
	readMessages  chan []byte
	writeMessages chan []byte
	closed        bool
}

func NewMockConn() *MockConn {
	return &MockConn{
		readMessages:  make(chan []byte, 10),
		writeMessages: make(chan []byte, 10),
	}
}

func (m *MockConn) ReadMessage() (messageType int, p []byte, err error) {
	msg, ok := <-m.readMessages
	if !ok {
		return 0, nil, websocket.ErrCloseSent
	}
	return websocket.TextMessage, msg, nil
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	if m.closed {
		return websocket.ErrCloseSent
	}
	m.writeMessages <- data
	return nil
}

func (m *MockConn) Close() error {
	m.closed = true
	close(m.writeMessages)
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadLimit(limit int64) {}

func (m *MockConn) SetPongHandler(h func(string) error) {}

func TestDeliverMessage(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Create auth service
	authService := auth.NewService(db)

	// Create server
	server := NewServer(db, authService, 10.0, 20)

	// Test broadcast message delivery
	t.Run("Broadcast Message Delivery", func(t *testing.T) {
		// Create two mock clients
		client1 := &Client{
			userID: "user1",
			conn:   &websocket.Conn{}, // This won't be used directly
			send:   make(chan []byte, 10),
			server: server,
		}
		client2 := &Client{
			userID: "user2",
			conn:   &websocket.Conn{}, // This won't be used directly
			send:   make(chan []byte, 10),
			server: server,
		}

		// Register the clients
		server.mu.Lock()
		server.clients["user1"] = client1
		server.clients["user2"] = client2
		server.mu.Unlock()

		// Create a broadcast message
		msg := models.Message{
			ID:          1,
			From:        "user1",
			Content:     "Hello everyone!",
			Timestamp:   time.Now(),
			IsBroadcast: true,
		}

		// Deliver the message
		if err := server.deliverMessage(msg); err != nil {
			t.Fatalf("Failed to deliver broadcast message: %v", err)
		}

		// Check that client2 received the message (client1 should not receive it as sender)
		select {
		case receivedMsg := <-client2.send:
			var decoded models.Message
			if err := json.Unmarshal(receivedMsg, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal received message: %v", err)
			}
			if decoded.From != "user1" || decoded.Content != "Hello everyone!" {
				t.Errorf("Message content mismatch: got %+v", decoded)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for broadcast message delivery")
		}

		// Check that client1 (sender) did not receive the message
		select {
		case <-client1.send:
			t.Error("Sender should not receive their own broadcast message")
		case <-time.After(100 * time.Millisecond):
			// This is the expected behavior
		}

		// Clean up
		server.mu.Lock()
		delete(server.clients, "user1")
		delete(server.clients, "user2")
		server.mu.Unlock()
	})

	// Test direct message delivery
	t.Run("Direct Message Delivery", func(t *testing.T) {
		// Set up mock for database update
		mock.ExpectExec(`^UPDATE messages SET status = \? WHERE id = \?$`).
			WithArgs("delivered", 2).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// mock.ExpectExec("UPDATE messages SET status = ? WHERE id = ?").
		// 	WithArgs("delivered", 2).
		// 	WillReturnResult(sqlmock.NewResult(0, 1))

		// Create two mock clients
		client1 := &Client{
			userID: "user1",
			conn:   &websocket.Conn{}, // This won't be used directly
			send:   make(chan []byte, 10),
			server: server,
		}
		client2 := &Client{
			userID: "user2",
			conn:   &websocket.Conn{}, // This won't be used directly
			send:   make(chan []byte, 10),
			server: server,
		}

		// Register the clients
		server.mu.Lock()
		server.clients["user1"] = client1
		server.clients["user2"] = client2
		server.mu.Unlock()

		// Create a direct message
		msg := models.Message{
			ID:        2,
			From:      "user1",
			To:        "user2",
			Content:   "Hello user2!",
			Timestamp: time.Now(),
		}

		// Deliver the message
		if err := server.deliverMessage(msg); err != nil {
			t.Fatalf("Failed to deliver direct message: %v", err)
		}

		// Check that client2 received the message
		select {
		case receivedMsg := <-client2.send:
			var decoded models.Message
			if err := json.Unmarshal(receivedMsg, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal received message: %v", err)
			}
			if decoded.From != "user1" || decoded.To != "user2" || decoded.Content != "Hello user2!" {
				t.Errorf("Message content mismatch: got %+v", decoded)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for direct message delivery")
		}

		// Verify that the message status was updated
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled database expectations: %s", err)
		}

		// Clean up
		server.mu.Lock()
		delete(server.clients, "user1")
		delete(server.clients, "user2")
		server.mu.Unlock()
	})

	// Test delivery to offline user
	t.Run("Offline User Message Delivery", func(t *testing.T) {
		// Create a message to offline user
		msg := models.Message{
			ID:        3,
			From:      "user1",
			To:        "offline_user",
			Content:   "Hello offline user!",
			Timestamp: time.Now(),
		}

		// Deliver the message (should not update database)
		if err := server.deliverMessage(msg); err != nil {
			t.Fatalf("Failed to deliver message to offline user: %v", err)
		}

		// No database update should be called
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled database expectations: %s", err)
		}
	})
}

func TestRetrieveUndeliveredMessages(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Create auth service
	authService := auth.NewService(db)

	// Create server
	server := NewServer(db, authService, 10.0, 20)

	// Mock client
	client := &Client{
		userID: "user1",
		conn:   &websocket.Conn{}, // This won't be used directly
		send:   make(chan []byte, 10),
		server: server,
	}

	// Register the client
	server.mu.Lock()
	server.clients["user1"] = client
	server.mu.Unlock()

	// Set up query results
	rows := sqlmock.NewRows([]string{
		"id", "from_user", "to_user", "timestamp", "content", "status", "is_broadcast", "signature",
	}).
		AddRow(1, "user2", "user1", time.Now(), "Direct message", "pending", false, "").
		AddRow(2, "user3", "broadcast", time.Now(), "Broadcast message", "pending", true, "")

		// Expect the query
	mock.ExpectQuery(`SELECT m\.id, m\.from_user, m\.to_user, m\.timestamp, m\.content, m\.status, m\.is_broadcast, m\.signature FROM messages m LEFT JOIN broadcast_deliveries bd ON m\.id = bd\.message_id AND bd\.user_id = \? WHERE \(\(m\.to_user = \? AND m\.status = 'pending'\) OR \(m\.is_broadcast = TRUE AND m\.status = 'pending'\)\) AND bd\.message_id IS NULL`).
		WithArgs("user1", "user1").
		WillReturnRows(rows)

	// Expect direct message status update with escaped question marks
	mock.ExpectExec(`^UPDATE messages SET status = \? WHERE id = \?$`).
		WithArgs("delivered", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect broadcast delivery recording with escaped characters.
	mock.ExpectExec(`^INSERT INTO broadcast_deliveries \(message_id, user_id\) VALUES \(\?, \?\)$`).
		WithArgs(2, "user1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// mock.ExpectQuery(`SELECT m\.id, m\.from_user, m\.to_user, m\.timestamp, m\.content, m\.status, m\.is_broadcast, m\.signature FROM messages m LEFT JOIN broadcast_deliveries bd ON m\.id = bd\.message_id AND bd\.user_id = \? WHERE \(\(m\.to_user = \? AND m\.status = 'pending'\) OR \(m\.is_broadcast = TRUE AND m\.status = 'pending'\)\) AND bd\.message_id IS NULL`).
	// 	WithArgs("user1", "user1").
	// 	WillReturnRows(rows)
	//
	// // Expect direct message status update
	// mock.ExpectExec("UPDATE messages SET status = ? WHERE id = ?").
	// 	WithArgs("delivered", 1).
	// 	WillReturnResult(sqlmock.NewResult(0, 1))
	//
	// // Expect broadcast delivery recording
	// mock.ExpectExec("INSERT INTO broadcast_deliveries").
	// 	WithArgs(2, "user1").
	// 	WillReturnResult(sqlmock.NewResult(0, 1))

	// Retrieve undelivered messages
	server.RetrieveUndeliveredMessages("user1")

	// Check that client received both messages
	receivedCount := 0
	timeout := time.After(time.Second)

messageLoop:
	for {
		select {
		case receivedMsg := <-client.send:
			var decoded models.Message
			if err := json.Unmarshal(receivedMsg, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal received message: %v", err)
			}
			receivedCount++
			if receivedCount == 2 {
				break messageLoop
			}
		case <-timeout:
			if receivedCount < 2 {
				t.Errorf("Expected 2 messages, received %d", receivedCount)
			}
			break messageLoop
		}
	}

	// Verify database expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled database expectations: %s", err)
	}

	// Clean up
	server.mu.Lock()
	delete(server.clients, "user1")
	server.mu.Unlock()
}

func TestHandleWebSocket(t *testing.T) {
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Create auth service
	authService := auth.NewService(db)

	// Create server
	server := NewServer(db, authService, 10.0, 20)

	// Test no token case
	t.Run("No Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ws", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		rr := httptest.NewRecorder()

		server.HandleWebSocket(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status unauthorized, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Unauthorized") {
			t.Errorf("Expected unauthorized message, got: %s", rr.Body.String())
		}
	})

	// Testing the websocket upgrade properly would require more complex setup
	// with mocked websocket library internals, which is beyond the scope of
	// a basic unit test. This would be better covered in an integration test.
}
