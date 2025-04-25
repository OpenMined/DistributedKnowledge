package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"websocketserver/auth"
	"websocketserver/models"
	"websocketserver/ws"
)

// DirectMessagePayload represents the JSON payload for direct messages via HTTP
type DirectMessagePayload struct {
	Type    string `json:"type"`
	Query   string `json:"query"`
	Message string `json:"message,omitempty"`
}

// HandleDirectMessage handles POST requests to send direct messages to users via websocket
func HandleDirectMessage(authService *auth.Service, wsServer *ws.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract and validate the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate the token
		claims, err := auth.ParseToken(tokenStr, authService)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract the user ID from the token claims
		fromUserID, ok := claims["user_id"].(string)
		if !ok || fromUserID == "" {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Read and parse the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var payload DirectMessagePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if payload.Type != "forward" || (payload.Query == "" && payload.Message == "") {
			http.Error(w, "Type must be 'forward' and either query or message field is required", http.StatusBadRequest)
			return
		}

		// Use query if provided, otherwise use message
		messageContent := payload.Query
		if messageContent == "" {
			messageContent = payload.Message
		}

		// Both fromUserID and toUserID will be the same value from token claims
		userID := fromUserID
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Create a forward message structure
		forwardMsg := struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    "forward",
			Message: messageContent,
		}

		// Marshal the forward message
		forwardMsgJSON, err := json.Marshal(forwardMsg)
		if err != nil {
			http.Error(w, "Error creating forward message", http.StatusInternalServerError)
			return
		}

		// Create a wrapper message
		wrapperMsg := struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    "forward",
			Message: string(forwardMsgJSON),
		}

		// Marshal the wrapper message
		content, err := json.Marshal(wrapperMsg)
		if err != nil {
			http.Error(w, "Error creating message content", http.StatusInternalServerError)
			return
		}

		// Create a websocket message
		wsMessage := models.Message{
			From:             fromUserID,
			To:               userID, // Send to self (same DK app)
			Timestamp:        time.Now(),
			Status:           "pending",
			Content:          string(content),
			IsForwardMessage: true,
		}

		// Register a response channel before sending the message
		responseCh := wsServer.RegisterResponseChannel(userID)
		defer wsServer.RemoveResponseChannel(userID) // Ensure cleanup

		// Set up a timeout context
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Try to deliver the message via the websocket server
		if err := wsServer.DeliverHTTPMessage(wsMessage); err != nil {
			log.Printf("Error delivering message to %s: %v", userID, err)
			http.Error(w, "Error delivering message", http.StatusInternalServerError)
			return
		}

		// Wait for response with timeout
		var responseMsg models.Message
		select {
		case responseMsg = <-responseCh:
			// Received response
			log.Printf("Received response for user %s", userID)
		case <-ctx.Done():
			http.Error(w, "Request timed out waiting for response", http.StatusGatewayTimeout)
			return
		}

		// Parse the response
		var responseWrapper struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(responseMsg.Content), &responseWrapper); err != nil {
			http.Error(w, "Error parsing response", http.StatusInternalServerError)
			return
		}

		var forwardResponse struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(responseWrapper.Message), &forwardResponse); err != nil {
			http.Error(w, "Error parsing forward response", http.StatusInternalServerError)
			return
		}

		// Return response to the HTTP client
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Success bool   `json:"success"`
			Answer  string `json:"answer"`
		}{
			Success: true,
			Answer:  forwardResponse.Message,
		}
		json.NewEncoder(w).Encode(response)
	}
}
