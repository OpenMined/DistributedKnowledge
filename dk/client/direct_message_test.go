package lib

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendDirectMessage(t *testing.T) {
	// Generate ed25519 key pair for testing
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ed25519 key: %v", err)
	}

	// Create a mock server to handle the direct message request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify the request path
		if r.URL.Path != "/direct-message/" {
			t.Errorf("Expected /direct-message/ path, got %s", r.URL.Path)
			http.Error(w, "Path not found", http.StatusNotFound)
			return
		}

		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			t.Errorf("Expected Authorization: Bearer test_token, got %s", authHeader)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode and verify the request payload
		var payload DirectMessagePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify payload fields
		if payload.Type != "forward" {
			t.Errorf("Expected Type 'forward', got '%s'", payload.Type)
		}
		if payload.Query != "test query" {
			t.Errorf("Expected Query 'test query', got '%s'", payload.Query)
		}
		if payload.Recipient != "recipient_id" {
			t.Errorf("Expected Recipient 'recipient_id', got '%s'", payload.Recipient)
		}

		// Return a successful response
		response := DirectMessageResponse{
			Success: true,
			Answer:  "This is the answer to your query",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client and set the token
	client := NewClient(server.URL, "test_user", privKey, pubKey)
	client.jwtToken = "test_token"

	// Send a direct message
	answer, err := client.SendDirectMessage("test query", "recipient_id")
	if err != nil {
		t.Fatalf("SendDirectMessage failed: %v", err)
	}

	// Verify the answer
	expectedAnswer := "This is the answer to your query"
	if answer != expectedAnswer {
		t.Errorf("Expected answer '%s', got '%s'", expectedAnswer, answer)
	}

	// Test error case: missing recipient
	_, err = client.SendDirectMessage("test query", "")
	if err == nil {
		t.Error("Expected error for missing recipient, got nil")
	}

	// Test convenience method for querying self
	// Create a new server that verifies recipient is the same as sender
	selfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload DirectMessagePayload
		json.NewDecoder(r.Body).Decode(&payload)
		
		// Verify payload fields
		if payload.Recipient != "test_user" {
			t.Errorf("Expected Recipient 'test_user', got '%s'", payload.Recipient)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response
		fmt.Fprintf(w, `{"success":true,"answer":"Self query answer"}`)
	}))
	defer selfServer.Close()

	// Create a client for self query test
	selfClient := NewClient(selfServer.URL, "test_user", privKey, pubKey)
	selfClient.jwtToken = "test_token"

	// Test QuerySelf
	selfAnswer, err := selfClient.QuerySelf("test self query")
	if err != nil {
		t.Fatalf("QuerySelf failed: %v", err)
	}
	if selfAnswer != "Self query answer" {
		t.Errorf("Expected self answer 'Self query answer', got '%s'", selfAnswer)
	}
}