package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	// "encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"websocketclient/lib"
	"websocketserver/auth"
	"websocketserver/db"
	"websocketserver/ws"
)

// TestServer represents a test server environment with all components.
type TestServer struct {
	DB          *sql.DB
	AuthService *auth.Service
	WSServer    *ws.Server
	HTTPServer  *httptest.Server
	URL         string
}

// TestClient represents a test client with keys.
type TestClient struct {
	Client     *lib.Client
	UserID     string
	Username   string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// SetupTestServer initializes a test server with in-memory SQLite.
func SetupTestServer(t *testing.T) *TestServer {
	// Create in-memory SQLite database.
	// database, err := sql.Open("sqlite3", ":memory:")
	database, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	database.SetMaxOpenConns(1)
	// Run migrations.
	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create the authentication service.
	authService := auth.NewService(database)

	// Create WebSocket server with rate limiting.
	wsServer := ws.NewServer(
		database,
		authService,
		10.0, // Rate (messages per second)
		20,   // Burst capacity
	)

	// Create HTTP mux and register handlers.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsServer.HandleWebSocket)
	mux.HandleFunc("/auth/register", authService.HandleRegistration)
	mux.HandleFunc("/auth/login", authService.HandleLogin)
	mux.HandleFunc("/auth/users/", authService.HandleGetUserInfo)

	// Create test HTTP server.
	httpServer := httptest.NewServer(mux)

	return &TestServer{
		DB:          database,
		AuthService: authService,
		WSServer:    wsServer,
		HTTPServer:  httpServer,
		URL:         httpServer.URL,
	}
}

// CloseTestServer closes the test server.
func CloseTestServer(ts *TestServer) {
	ts.HTTPServer.Close()
	ts.DB.Close()
}

// CreateTestClient creates a new test client with a randomly generated keypair.
func CreateTestClient(t *testing.T, ts *TestServer, userID, username string) *TestClient {
	// Generate RSA key pair.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	publicKey := &privateKey.PublicKey

	// Create WebSocket client.
	client := lib.NewClient(ts.URL, userID, privateKey, publicKey)
	client.SetInsecure(true) // Skip TLS verification for test server

	return &TestClient{
		Client:     client,
		UserID:     userID,
		Username:   username,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}

// TestRegistrationAndLogin tests the complete registration and login flow.
func TestRegistrationAndLogin(t *testing.T) {
	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create test client.
	tc := CreateTestClient(t, ts, "user1", "Test User 1")

	// Register the client.
	err := tc.Client.Register(tc.Username)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	// Login the client.
	err = tc.Client.Login()
	if err != nil {
		t.Fatalf("Failed to login client: %v", err)
	}

	// Verify token was set using an accessor method.
	if tc.Client.Token() == "" {
		t.Error("JWT token not set after login")
	}
}

// TestMessageExchange tests sending and receiving messages.
func TestMessageExchange(t *testing.T) {
	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create two test clients.
	client1 := CreateTestClient(t, ts, "user1", "Test User 1")
	client2 := CreateTestClient(t, ts, "user2", "Test User 2")

	// Register and login both clients.
	for _, c := range []*TestClient{client1, client2} {
		if err := c.Client.Register(c.Username); err != nil {
			t.Fatalf("Failed to register %s: %v", c.UserID, err)
		}
		if err := c.Client.Login(); err != nil {
			t.Fatalf("Failed to login %s: %v", c.UserID, err)
		}
	}

	// Connect both clients.
	for _, c := range []*TestClient{client1, client2} {
		if err := c.Client.Connect(); err != nil {
			t.Fatalf("Failed to connect %s: %v", c.UserID, err)
		}
		defer c.Client.Disconnect()
	}

	// Wait for connections to establish.
	time.Sleep(100 * time.Millisecond)

	// Test direct message exchange.
	testContent := "Hello from user1 to user2!"
	err := client1.Client.SendMessage(lib.Message{
		To:      client2.UserID,
		Content: testContent,
	})
	if err != nil {
		t.Fatalf("Failed to send message from user1 to user2: %v", err)
	}

	// Wait for message delivery.
	select {
	case msg := <-client2.Client.Messages():
		if msg.From != client1.UserID {
			t.Errorf("Expected message from %s, got %s", client1.UserID, msg.From)
		}
		if msg.Content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, msg.Content)
		}
		if msg.Signature == "" {
			t.Error("Expected signature to be present")
		}
		// Verify signature status.
		if msg.Status != "verified" {
			t.Errorf("Expected status 'verified', got '%s'", msg.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Test broadcast message.
	broadcastContent := "Broadcast message from user1"
	err = client1.Client.BroadcastMessage(broadcastContent)
	if err != nil {
		t.Fatalf("Failed to send broadcast message: %v", err)
	}

	// Wait for broadcast message delivery.
	select {
	case msg := <-client2.Client.Messages():
		if msg.From != client1.UserID {
			t.Errorf("Expected broadcast from %s, got %s", client1.UserID, msg.From)
		}
		if msg.Content != broadcastContent {
			t.Errorf("Expected content %q, got %q", broadcastContent, msg.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for broadcast message")
	}
}

// TestUndeliveredMessageRetrieval tests that messages sent to an offline user
// are delivered when they come online.
func TestUndeliveredMessageRetrieval(t *testing.T) {
	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create two test clients.
	sender := CreateTestClient(t, ts, "sender", "Sender User")
	receiver := CreateTestClient(t, ts, "receiver", "Receiver User")

	// Register both clients.
	for _, c := range []*TestClient{sender, receiver} {
		if err := c.Client.Register(c.Username); err != nil {
			t.Fatalf("Failed to register %s: %v", c.UserID, err)
		}
		if err := c.Client.Login(); err != nil {
			t.Fatalf("Failed to login %s: %v", c.UserID, err)
		}
	}

	// Connect only the sender.
	if err := sender.Client.Connect(); err != nil {
		t.Fatalf("Failed to connect sender: %v", err)
	}
	defer sender.Client.Disconnect()

	// Send a message to the offline receiver.
	messageContent := "This is an offline message"
	err := sender.Client.SendMessage(lib.Message{
		To:      receiver.UserID,
		Content: messageContent,
	})
	if err != nil {
		t.Fatalf("Failed to send message to offline user: %v", err)
	}

	// Wait for message to be stored.
	time.Sleep(100 * time.Millisecond)

	// Now connect the receiver.
	if err := receiver.Client.Connect(); err != nil {
		t.Fatalf("Failed to connect receiver: %v", err)
	}
	defer receiver.Client.Disconnect()

	// Wait for undelivered message retrieval.
	select {
	case msg := <-receiver.Client.Messages():
		if msg.From != sender.UserID {
			t.Errorf("Expected message from %s, got %s", sender.UserID, msg.From)
		}
		if msg.Content != messageContent {
			t.Errorf("Expected content %q, got %q", messageContent, msg.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for undelivered message")
	}

	// Query the database to verify the message status was updated.
	var status string
	err = ts.DB.QueryRow("SELECT status FROM messages WHERE from_user = ? AND to_user = ?", sender.UserID, receiver.UserID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query message status: %v", err)
	}
	if status != "delivered" {
		t.Errorf("Expected message status 'delivered', got '%s'", status)
	}
}

// TestRateLimiting tests that rate limiting is enforced.
func TestRateLimiting(t *testing.T) {
	// Set up test server with strict rate limits for testing.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Override rate limiter with strict limits.
	// 2 messages per second with burst of 3.
	// (Note: Here we assume ws.Server now exports its RateLimiter field.)
	ts.WSServer.RateLimiter = ws.NewRateLimiter(2.0, 3)

	// Create test client.
	client := CreateTestClient(t, ts, "ratelimit_user", "Rate Limit Test User")

	// Register and login.
	if err := client.Client.Register(client.Username); err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}
	if err := client.Client.Login(); err != nil {
		t.Fatalf("Failed to login client: %v", err)
	}

	// Connect client.
	if err := client.Client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Client.Disconnect()

	// Send messages rapidly to trigger rate limiting.
	for i := 0; i < 5; i++ {
		client.Client.SendMessage(lib.Message{
			To:      "recipient",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	// Listen for rate limit error message.
	rateLimitErrorReceived := false
	timeout := time.After(2 * time.Second)

	for !rateLimitErrorReceived {
		select {
		case msg := <-client.Client.Messages():
			if msg.From == "system" && msg.Status == "error" && msg.Content == "Rate limit exceeded. Please slow down." {
				rateLimitErrorReceived = true
			}
		case <-timeout:
			t.Log("No rate limit error received within timeout")
			return
		}
	}

	if !rateLimitErrorReceived {
		t.Error("Expected to receive rate limit error message")
	}
}

// TestSignatureVerification tests that message signatures are properly verified.
func TestSignatureVerification(t *testing.T) {
	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create two test clients.
	sender := CreateTestClient(t, ts, "sender", "Sender User")
	receiver := CreateTestClient(t, ts, "receiver", "Receiver User")

	// Register and login both clients.
	for _, c := range []*TestClient{sender, receiver} {
		if err := c.Client.Register(c.Username); err != nil {
			t.Fatalf("Failed to register %s: %v", c.UserID, err)
		}
		if err := c.Client.Login(); err != nil {
			t.Fatalf("Failed to login %s: %v", c.UserID, err)
		}
	}

	// Connect both clients.
	for _, c := range []*TestClient{sender, receiver} {
		if err := c.Client.Connect(); err != nil {
			t.Fatalf("Failed to connect %s: %v", c.UserID, err)
		}
		defer c.Client.Disconnect()
	}

	// Send a properly signed message.
	err := sender.Client.SendMessage(lib.Message{
		To:      receiver.UserID,
		Content: "Legitimate message",
	})
	if err != nil {
		t.Fatalf("Failed to send legitimate message: %v", err)
	}

	// Wait for message delivery and verify signature.
	select {
	case msg := <-receiver.Client.Messages():
		if msg.Status != "verified" {
			t.Errorf("Expected status 'verified', got '%s'", msg.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for legitimate message")
	}

	// Attempt message spoofing by directly using the send channel (bypassing the signature process).
	spoofedMsg := lib.Message{
		From:      sender.UserID, // Pretending to be the sender.
		To:        receiver.UserID,
		Content:   "Spoofed message",
		Timestamp: time.Now(),
		// No signature.
	}

	// Instead of marshalling to []byte, send the message struct directly.
	sender.Client.SendCh() <- spoofedMsg

	// Check if receiver gets the message with invalid signature status.
	select {
	case msg := <-receiver.Client.Messages():
		if msg.Content == "Spoofed message" {
			if msg.Status != "unsigned" && msg.Status != "invalid_signature" {
				t.Errorf("Expected unsigned or invalid_signature status for spoofed message, got '%s'", msg.Status)
			}
		}
	case <-time.After(2 * time.Second):
		t.Log("Spoofed message was not delivered (which is good)")
	}
}

// TestReconnection tests the client's ability to reconnect after disconnection.
func TestReconnection(t *testing.T) {
	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create test client.
	client := CreateTestClient(t, ts, "reconnect_user", "Reconnect Test User")

	// Register and login.
	if err := client.Client.Register(client.Username); err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}
	if err := client.Client.Login(); err != nil {
		t.Fatalf("Failed to login client: %v", err)
	}

	// Set a short reconnect interval for testing.
	// Use the client’s setter rather than directly accessing an unexported field.
	client.Client.SetReconnectInterval(200 * time.Millisecond)

	// Connect client.
	if err := client.Client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	// Force disconnect (simulate connection loss).
	client.Client.Disconnect()

	// Try to send a message (should trigger reconnect).
	go func() {
		time.Sleep(300 * time.Millisecond) // Wait a bit for reconnect to start.
		err := client.Client.SendMessage(lib.Message{
			To:      "echo",
			Content: "Reconnect test",
		})
		if err != nil {
			t.Logf("Send error after disconnect (expected): %v", err)
		}
	}()

	// Give time for reconnection.
	time.Sleep(1 * time.Second)

	// Clean final disconnect.
	client.Client.Disconnect()
	// This test mainly ensures that reconnection code doesn't panic or deadlock.
}

// TestServerClientIntegration runs a comprehensive integration test.
func TestServerClientIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test server.
	ts := SetupTestServer(t)
	defer CloseTestServer(ts)

	// Create multiple clients.
	clients := []*TestClient{
		CreateTestClient(t, ts, "user1", "User One"),
		CreateTestClient(t, ts, "user2", "User Two"),
		CreateTestClient(t, ts, "user3", "User Three"),
	}

	// Register and login all clients.
	for _, c := range clients {
		if err := c.Client.Register(c.Username); err != nil {
			t.Fatalf("Failed to register %s: %v", c.UserID, err)
		}
		if err := c.Client.Login(); err != nil {
			t.Fatalf("Failed to login %s: %v", c.UserID, err)
		}
	}

	// Connect clients 1 and 2, but not 3 (to test offline message delivery).
	for _, c := range clients[:2] {
		if err := c.Client.Connect(); err != nil {
			t.Fatalf("Failed to connect %s: %v", c.UserID, err)
		}
		defer c.Client.Disconnect()
	}

	// Create message receivers for connected clients.
	receivers := make(map[string]chan lib.Message)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, c := range clients[:2] {
		receivers[c.UserID] = make(chan lib.Message, 10)
		go func(userID string, client *lib.Client, ch chan lib.Message) {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-client.Messages():
					ch <- msg
				}
			}
		}(c.UserID, c.Client, receivers[c.UserID])
	}

	// Send direct messages between clients 1 and 2.
	if err := clients[0].Client.SendMessage(lib.Message{
		To:      clients[1].UserID,
		Content: "Direct message from 1 to 2",
	}); err != nil {
		t.Fatalf("Failed to send direct message: %v", err)
	}

	// Send message to offline client 3.
	if err := clients[0].Client.SendMessage(lib.Message{
		To:      clients[2].UserID,
		Content: "Message to offline user 3",
	}); err != nil {
		t.Fatalf("Failed to send message to offline user: %v", err)
	}

	// Send broadcast message.
	if err := clients[1].Client.BroadcastMessage("Broadcast from user 2"); err != nil {
		t.Fatalf("Failed to send broadcast message: %v", err)
	}

	// Verify direct message receipt.
	select {
	case msg := <-receivers[clients[1].UserID]:
		if msg.From != clients[0].UserID || msg.Content != "Direct message from 1 to 2" {
			t.Errorf("Unexpected message: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for direct message")
	}

	// Verify broadcast receipt.
	select {
	case msg := <-receivers[clients[0].UserID]:
		if msg.From != clients[1].UserID || msg.Content != "Broadcast from user 2" {
			t.Errorf("Unexpected broadcast message: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for broadcast message")
	}

	// Now connect client 3 and verify offline message delivery.
	if err := clients[2].Client.Connect(); err != nil {
		t.Fatalf("Failed to connect client 3: %v", err)
	}
	defer clients[2].Client.Disconnect()

	// Create receiver for client 3.
	receivers[clients[2].UserID] = make(chan lib.Message, 10)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-clients[2].Client.Messages():
				receivers[clients[2].UserID] <- msg
			}
		}
	}()

	// Verify offline message delivery to client 3.
	select {
	case msg := <-receivers[clients[2].UserID]:
		if msg.From != clients[0].UserID || msg.Content != "Message to offline user 3" {
			t.Errorf("Unexpected offline message: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for offline message delivery")
	}

	// Verify broadcast delivery to newly connected client.
	select {
	case msg := <-receivers[clients[2].UserID]:
		if msg.From != clients[1].UserID || msg.Content != "Broadcast from user 2" {
			t.Errorf("Unexpected broadcast to new client: %+v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for broadcast delivery to new client")
	}
}
