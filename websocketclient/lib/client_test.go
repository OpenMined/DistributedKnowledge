package lib

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Generate ed25519 key pair for testing.
	// Note: ed25519.GenerateKey returns (publicKey, privateKey, error).
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	serverURL := "https://example.com"
	userID := "test_user"

	client := NewClient(serverURL, userID, privKey, pubKey)

	if client.serverURL != serverURL {
		t.Errorf("Expected serverURL %s, got %s", serverURL, client.serverURL)
	}
	if client.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, client.UserID)
	}
	// Compare byte slices using bytes.Equal.
	if !bytes.Equal(client.privateKey, privKey) {
		t.Error("Expected private key to match")
	}
	if !bytes.Equal(client.publicKey, pubKey) {
		t.Error("Expected public key to match")
	}

	// Verify channels are created.
	if client.recvCh == nil {
		t.Error("Receive channel not initialized")
	}
	if client.sendCh == nil {
		t.Error("Send channel not initialized")
	}
	if client.doneCh == nil {
		t.Error("Done channel not initialized")
	}

	// Verify own public key is in cache.
	if cachedKey, exists := client.pubKeyCache[userID]; !exists || !bytes.Equal(cachedKey, pubKey) {
		t.Error("Client's own public key not cached properly")
	}
}

func TestSignAndVerifyMessage(t *testing.T) {
	// Generate ed25519 key pair for testing.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	client := NewClient("https://example.com", "test_user", privKey, pubKey)

	// Create a test message.
	msg := Message{
		From:    "test_user",
		To:      "recipient",
		Content: "Hello, this is a test message!",
	}

	// Sign the message.
	err = client.signMessage(&msg)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify signature is not empty.
	if msg.Signature == "" {
		t.Error("Signature is empty after signing")
	}

	// Check that the signature decodes correctly from base64.
	_, err = base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		t.Errorf("Signature is not valid base64: %v", err)
	}

	// Verify the signature with the public key.
	isValid := client.verifyMessageSignature(msg, pubKey)
	if !isValid {
		t.Error("Signature verification failed")
	}

	// Test with a tampered message.
	tamperedMsg := msg
	tamperedMsg.Content = "Tampered content"
	isValid = client.verifyMessageSignature(tamperedMsg, pubKey)
	if isValid {
		t.Error("Signature verification should fail with tampered message")
	}
}

func TestRegister(t *testing.T) {
	// Generate ed25519 key pair for testing.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method and path.
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/auth/register" {
			t.Errorf("Expected /auth/register path, got %s", r.URL.Path)
		}

		// Parse the request body.
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify payload.
		if payload["user_id"] != "test_user" {
			t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
		}
		if payload["username"] != "Test User" {
			t.Errorf("Expected username 'Test User', got '%s'", payload["username"])
		}

		// Instead of a PEM check, verify that the public_key is a valid base64 string with the expected length.
		pubKeyBytes, err := base64.StdEncoding.DecodeString(payload["public_key"])
		if err != nil {
			t.Errorf("public_key is not properly base64 encoded: %v", err)
		}
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
		}

		// Return success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("User registered successfully"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_user", privKey, pubKey)

	// Register the client.
	err = client.Register("Test User")
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
}

func TestLogin(t *testing.T) {
	// Generate ed25519 key pair for testing.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	// Create a challenge for testing.
	challenge := "test_challenge"
	challengeBase64 := base64.StdEncoding.EncodeToString([]byte(challenge))

	// Track request count to serve different responses.
	requestCount := 0

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		requestCount++
		if requestCount == 1 {
			// First request: challenge request.
			if r.URL.Path != "/auth/login" {
				t.Errorf("Expected /auth/login path, got %s", r.URL.Path)
			}

			// Parse request body.
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Verify payload.
			if payload["user_id"] != "test_user" {
				t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
			}

			// Return the challenge.
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"challenge":"%s"}`, challengeBase64)
		} else {
			// Second request: verification.
			if r.URL.Path != "/auth/login" || r.URL.Query().Get("verify") != "true" {
				t.Errorf("Expected /auth/login?verify=true path, got %s", r.URL.String())
			}

			// Parse request body.
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Verify payload.
			if payload["user_id"] != "test_user" {
				t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
			}
			if payload["signature"] == "" {
				t.Error("Expected signature, got empty string")
			}

			// Return success with token.
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"token":"test_token"}`)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_user", privKey, pubKey)

	// Perform login.
	err = client.Login()
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Verify that the token was set.
	if client.jwtToken != "test_token" {
		t.Errorf("Expected token 'test_token', got '%s'", client.jwtToken)
	}
}

func TestGetUserPublicKey(t *testing.T) {
	// Generate ed25519 key pair for testing.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	// Instead of a PEM, encode the public key as base64.
	publicKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	// Create a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		expectedPath := "/auth/users/other_user"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s path, got %s", expectedPath, r.URL.Path)
		}

		// Return the public key.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"user_id":"other_user","public_key":%q}`, publicKeyB64)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_user", privKey, pubKey)
	client.jwtToken = "test_token" // Set a dummy token.

	// Get the public key for another user.
	otherKey, err := client.GetUserPublicKey("other_user")
	if err != nil {
		t.Fatalf("Failed to get user public key: %v", err)
	}

	// Verify that the key matches the original.
	if !bytes.Equal(otherKey, pubKey) {
		t.Error("Retrieved public key doesn't match the expected one")
	}

	// Verify that the key is cached.
	client.pubKeyCacheMu.RLock()
	cachedKey, exists := client.pubKeyCache["other_user"]
	client.pubKeyCacheMu.RUnlock()
	if !exists {
		t.Error("Public key not cached")
	}
	if !bytes.Equal(cachedKey, pubKey) {
		t.Error("Cached key doesn't match the expected one")
	}

	// Get the key again (should come from the cache).
	cachedKeyResult, err := client.GetUserPublicKey("other_user")
	if err != nil {
		t.Fatalf("Failed to get cached user public key: %v", err)
	}
	if !bytes.Equal(cachedKeyResult, pubKey) {
		t.Error("Cached key result doesn't match the expected one")
	}
}

func TestSendMessage(t *testing.T) {
	// Generate ed25519 key pair for testing.
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	client := NewClient("https://example.com", "test_user", privKey, pubKey)

	// Test sending a direct message.
	msg := Message{
		To:      "recipient",
		Content: "Hello, this is a test message!",
	}

	// Spin a goroutine to receive from the send channel to prevent blocking.
	go func() {
		<-client.sendCh
	}()

	err = client.SendMessage(msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Test sending a broadcast message.
	go func() {
		sentMsg := <-client.sendCh

		// Verify broadcast message properties.
		if sentMsg.From != "test_user" {
			t.Errorf("Expected From to be 'test_user', got '%s'", sentMsg.From)
		}
		if sentMsg.To != "broadcast" {
			t.Errorf("Expected To to be 'broadcast', got '%s'", sentMsg.To)
		}
		if sentMsg.Content != "Broadcast test" {
			t.Errorf("Expected Content to be 'Broadcast test', got '%s'", sentMsg.Content)
		}
		if sentMsg.Signature == "" {
			t.Error("Expected Signature to be set")
		}
	}()

	err = client.BroadcastMessage("Broadcast test")
	if err != nil {
		t.Fatalf("Failed to send broadcast message: %v", err)
	}
}

// package lib
//
// import (
// 	"crypto/rand"
// 	"crypto/rsa"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"
// )
//
// func TestNewClient(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	serverURL := "https://example.com"
// 	userID := "test_user"
//
// 	client := NewClient(serverURL, userID, privateKey, publicKey)
//
// 	if client.serverURL != serverURL {
// 		t.Errorf("Expected serverURL %s, got %s", serverURL, client.serverURL)
// 	}
// 	if client.UserID != userID {
// 		t.Errorf("Expected userID %s, got %s", userID, client.UserID)
// 	}
// 	if client.privateKey != privateKey {
// 		t.Errorf("Expected private key reference to match")
// 	}
// 	if client.publicKey != publicKey {
// 		t.Errorf("Expected public key reference to match")
// 	}
//
// 	// Verify channels are created
// 	if client.recvCh == nil {
// 		t.Error("Receive channel not initialized")
// 	}
// 	if client.sendCh == nil {
// 		t.Error("Send channel not initialized")
// 	}
// 	if client.doneCh == nil {
// 		t.Error("Done channel not initialized")
// 	}
//
// 	// Verify own public key is in cache
// 	if cachedKey, exists := client.pubKeyCache[userID]; !exists || cachedKey != publicKey {
// 		t.Error("Client's own public key not cached properly")
// 	}
// }
//
// func TestSignAndVerifyMessage(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	client := NewClient("https://example.com", "test_user", privateKey, publicKey)
//
// 	// Create a test message
// 	msg := Message{
// 		From:    "test_user",
// 		To:      "recipient",
// 		Content: "Hello, this is a test message!",
// 	}
//
// 	// Sign the message
// 	err = client.signMessage(&msg)
// 	if err != nil {
// 		t.Fatalf("Failed to sign message: %v", err)
// 	}
//
// 	// Verify signature is not empty
// 	if msg.Signature == "" {
// 		t.Error("Signature is empty after signing")
// 	}
//
// 	// Decode the signature to verify it's valid base64
// 	_, err = base64.StdEncoding.DecodeString(msg.Signature)
// 	if err != nil {
// 		t.Errorf("Signature is not valid base64: %v", err)
// 	}
//
// 	// Verify the signature with the public key
// 	isValid := client.verifyMessageSignature(msg, publicKey)
// 	if !isValid {
// 		t.Error("Signature verification failed")
// 	}
//
// 	// Test with tampered message
// 	tamperedMsg := msg
// 	tamperedMsg.Content = "Tampered content"
// 	isValid = client.verifyMessageSignature(tamperedMsg, publicKey)
// 	if isValid {
// 		t.Error("Signature verification should fail with tampered message")
// 	}
// }
//
// func TestPublicKeyToPEM(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	// Convert public key to PEM
// 	pemString, err := PublicKeyToPEM(publicKey)
// 	if err != nil {
// 		t.Fatalf("Failed to convert public key to PEM: %v", err)
// 	}
//
// 	// Verify it's a valid PEM format
// 	if !strings.Contains(pemString, "-----BEGIN PUBLIC KEY-----") {
// 		t.Error("PEM string doesn't contain the expected header")
// 	}
// 	if !strings.Contains(pemString, "-----END PUBLIC KEY-----") {
// 		t.Error("PEM string doesn't contain the expected footer")
// 	}
//
// 	// Parse the PEM back to public key
// 	parsedKey, err := ParseRSAPublicKeyFromPEM(pemString)
// 	if err != nil {
// 		t.Fatalf("Failed to parse PEM back to public key: %v", err)
// 	}
//
// 	// Verify the parsed key matches the original
// 	if parsedKey.N.Cmp(publicKey.N) != 0 || parsedKey.E != publicKey.E {
// 		t.Error("Parsed public key doesn't match the original")
// 	}
// }
//
// func TestRegister(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Check method and path
// 		if r.Method != "POST" {
// 			t.Errorf("Expected POST request, got %s", r.Method)
// 		}
// 		if r.URL.Path != "/auth/register" {
// 			t.Errorf("Expected /auth/register path, got %s", r.URL.Path)
// 		}
//
// 		// Parse request body
// 		var payload map[string]string
// 		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
// 			t.Errorf("Failed to decode request body: %v", err)
// 			http.Error(w, "Bad request", http.StatusBadRequest)
// 			return
// 		}
//
// 		// Verify payload
// 		if payload["user_id"] != "test_user" {
// 			t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
// 		}
// 		if payload["username"] != "Test User" {
// 			t.Errorf("Expected username 'Test User', got '%s'", payload["username"])
// 		}
// 		if !strings.Contains(payload["public_key"], "-----BEGIN PUBLIC KEY-----") {
// 			t.Errorf("Public key not in PEM format: %s", payload["public_key"])
// 		}
//
// 		// Return success
// 		w.WriteHeader(http.StatusCreated)
// 		w.Write([]byte("User registered successfully"))
// 	}))
// 	defer server.Close()
//
// 	client := NewClient(server.URL, "test_user", privateKey, publicKey)
//
// 	// Register the client
// 	err = client.Register("Test User")
// 	if err != nil {
// 		t.Fatalf("Failed to register: %v", err)
// 	}
// }
//
// func TestLogin(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	// Create a challenge for testing
// 	challenge := "test_challenge"
// 	challengeBase64 := base64.StdEncoding.EncodeToString([]byte(challenge))
//
// 	// Track request count to serve different responses
// 	requestCount := 0
//
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != "POST" {
// 			t.Errorf("Expected POST request, got %s", r.Method)
// 			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 			return
// 		}
//
// 		requestCount++
// 		if requestCount == 1 {
// 			// First request: challenge request
// 			if r.URL.Path != "/auth/login" {
// 				t.Errorf("Expected /auth/login path, got %s", r.URL.Path)
// 			}
//
// 			// Parse request body
// 			var payload map[string]string
// 			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
// 				t.Errorf("Failed to decode request body: %v", err)
// 				http.Error(w, "Bad request", http.StatusBadRequest)
// 				return
// 			}
//
// 			// Verify payload
// 			if payload["user_id"] != "test_user" {
// 				t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
// 			}
//
// 			// Return challenge
// 			w.Header().Set("Content-Type", "application/json")
// 			fmt.Fprintf(w, `{"challenge":"%s"}`, challengeBase64)
// 		} else {
// 			// Second request: verification
// 			if r.URL.Path != "/auth/login" || r.URL.Query().Get("verify") != "true" {
// 				t.Errorf("Expected /auth/login?verify=true path, got %s", r.URL.String())
// 			}
//
// 			// Parse request body
// 			var payload map[string]string
// 			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
// 				t.Errorf("Failed to decode request body: %v", err)
// 				http.Error(w, "Bad request", http.StatusBadRequest)
// 				return
// 			}
//
// 			// Verify payload
// 			if payload["user_id"] != "test_user" {
// 				t.Errorf("Expected user_id 'test_user', got '%s'", payload["user_id"])
// 			}
// 			if payload["signature"] == "" {
// 				t.Error("Expected signature, got empty string")
// 			}
//
// 			// In a real test, we'd verify the signature, but that's complex
// 			// Return success with token
// 			w.Header().Set("Content-Type", "application/json")
// 			fmt.Fprint(w, `{"token":"test_token"}`)
// 		}
// 	}))
// 	defer server.Close()
//
// 	client := NewClient(server.URL, "test_user", privateKey, publicKey)
//
// 	// Login
// 	err = client.Login()
// 	if err != nil {
// 		t.Fatalf("Failed to login: %v", err)
// 	}
//
// 	// Verify token was set
// 	if client.jwtToken != "test_token" {
// 		t.Errorf("Expected token 'test_token', got '%s'", client.jwtToken)
// 	}
// }
//
// func TestGetUserPublicKey(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	// Create test PEM for server response
// 	publicKeyPEM, err := PublicKeyToPEM(publicKey)
// 	if err != nil {
// 		t.Fatalf("Failed to convert public key to PEM: %v", err)
// 	}
//
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != "GET" {
// 			t.Errorf("Expected GET request, got %s", r.Method)
// 			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 			return
// 		}
//
// 		expectedPath := "/auth/users/other_user"
// 		if r.URL.Path != expectedPath {
// 			t.Errorf("Expected %s path, got %s", expectedPath, r.URL.Path)
// 		}
//
// 		// Return public key
// 		w.Header().Set("Content-Type", "application/json")
// 		fmt.Fprintf(w, `{"user_id":"other_user","public_key":%q}`, publicKeyPEM)
// 	}))
// 	defer server.Close()
//
// 	client := NewClient(server.URL, "test_user", privateKey, publicKey)
// 	client.jwtToken = "test_token" // Set token
//
// 	// Get public key for another user
// 	otherKey, err := client.GetUserPublicKey("other_user")
// 	if err != nil {
// 		t.Fatalf("Failed to get user public key: %v", err)
// 	}
//
// 	// Verify key matches original
// 	if otherKey.N.Cmp(publicKey.N) != 0 || otherKey.E != publicKey.E {
// 		t.Error("Retrieved public key doesn't match the expected one")
// 	}
//
// 	// Verify key is cached
// 	client.pubKeyCacheMu.RLock()
// 	cachedKey, exists := client.pubKeyCache["other_user"]
// 	client.pubKeyCacheMu.RUnlock()
// 	if !exists {
// 		t.Error("Public key not cached")
// 	}
// 	if cachedKey.N.Cmp(publicKey.N) != 0 || cachedKey.E != publicKey.E {
// 		t.Error("Cached key doesn't match the expected one")
// 	}
//
// 	// Get key again (should use cache)
// 	cachedKeyResult, err := client.GetUserPublicKey("other_user")
// 	if err != nil {
// 		t.Fatalf("Failed to get cached user public key: %v", err)
// 	}
// 	if cachedKeyResult.N.Cmp(publicKey.N) != 0 || cachedKeyResult.E != publicKey.E {
// 		t.Error("Cached key result doesn't match the expected one")
// 	}
// }
//
// func TestSendMessage(t *testing.T) {
// 	// Generate RSA key pair for testing
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		t.Fatalf("Failed to generate RSA key: %v", err)
// 	}
// 	publicKey := &privateKey.PublicKey
//
// 	client := NewClient("https://example.com", "test_user", privateKey, publicKey)
//
// 	// Test sending a direct message
// 	msg := Message{
// 		To:      "recipient",
// 		Content: "Hello, this is a test message!",
// 	}
//
// 	go func() {
// 		// Read from send channel to prevent blocking
// 		<-client.sendCh
// 	}()
//
// 	err = client.SendMessage(msg)
// 	if err != nil {
// 		t.Fatalf("Failed to send message: %v", err)
// 	}
//
// 	// Test sending a broadcast message
// 	go func() {
// 		sentMsg := <-client.sendCh
//
// 		// Verify broadcast message properties
// 		if sentMsg.From != "test_user" {
// 			t.Errorf("Expected From to be 'test_user', got '%s'", sentMsg.From)
// 		}
// 		if sentMsg.To != "broadcast" {
// 			t.Errorf("Expected To to be 'broadcast', got '%s'", sentMsg.To)
// 		}
// 		if sentMsg.Content != "Broadcast test" {
// 			t.Errorf("Expected Content to be 'Broadcast test', got '%s'", sentMsg.Content)
// 		}
// 		if sentMsg.Signature == "" {
// 			t.Error("Expected Signature to be set")
// 		}
// 	}()
//
// 	err = client.BroadcastMessage("Broadcast test")
// 	if err != nil {
// 		t.Fatalf("Failed to send broadcast message: %v", err)
// 	}
// }
