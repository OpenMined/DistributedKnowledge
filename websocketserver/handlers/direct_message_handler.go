package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
	// Removed recipient field - will use token owner as recipient
}

// AuthenticationResult holds the result of token authentication
type AuthenticationResult struct {
	UserID    string
	Valid     bool
	ErrorMsg  string
	ErrorCode int
}

// authenticateRequest validates the Authorization header and JWT token
// Also logs security events for auditing
func authenticateRequest(r *http.Request, authService *auth.Service) AuthenticationResult {
	result := AuthenticationResult{
		Valid:     false,
		ErrorCode: http.StatusUnauthorized,
	}

	clientIP := auth.GetClientIP(r)
	securityLogger := auth.NewLogger()

	// Extract and validate the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		result.ErrorMsg = "Missing or invalid Authorization header"

		// Log security event
		securityLogger.LogAuthEvent(auth.SecurityEvent{
			Timestamp: time.Now(),
			Event:     auth.EventUnauthorizedAccess,
			UserID:    "unknown",
			IP:        clientIP,
			Success:   false,
			Details:   "Missing or invalid Authorization header",
		})

		return result
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	// Use our enhanced token verification
	tokenResult := auth.VerifyToken(tokenStr, authService, "")
	if !tokenResult.Valid || tokenResult.Error != nil {
		result.ErrorMsg = fmt.Sprintf("Invalid token: %v", tokenResult.Error)

		// Log security event with user ID from token if available
		userID := "unknown"
		if tokenResult.UserID != "" {
			userID = tokenResult.UserID
		}

		securityLogger.LogAuthEvent(auth.SecurityEvent{
			Timestamp: time.Now(),
			Event:     auth.EventTokenVerification,
			UserID:    userID,
			IP:        clientIP,
			Success:   false,
			Details:   fmt.Sprintf("Token verification failed: %v", tokenResult.Error),
		})

		return result
	}

	// Token is valid, extract user ID
	result.UserID = tokenResult.UserID
	result.Valid = true

	// Log successful authentication
	securityLogger.LogAuthEvent(auth.SecurityEvent{
		Timestamp: time.Now(),
		Event:     auth.EventTokenVerification,
		UserID:    result.UserID,
		IP:        clientIP,
		Success:   true,
		Details:   "Token successfully validated",
	})

	return result
}

// HandleDirectMessage handles POST requests to send direct messages to users via websocket
func HandleDirectMessage(authService *auth.Service, wsServer *ws.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		securityLogger := auth.NewLogger()
		clientIP := auth.GetClientIP(r)

		// Only allow POST requests
		if r.Method != http.MethodPost {
			auth.SendAuthErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Authenticate the request
		authResult := authenticateRequest(r, authService)
		if !authResult.Valid {
			auth.SendAuthErrorResponse(w, authResult.ErrorMsg, authResult.ErrorCode)
			return
		}

		// Get authenticated user ID
		fromUserID := authResult.UserID

		// Read and parse the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Failed to read request body: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var payload DirectMessagePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Invalid JSON payload: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Validate message type
		if payload.Type != "forward" {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   "Invalid message type: must be 'forward'",
			})
			auth.SendAuthErrorResponse(w, "Type must be 'forward'", http.StatusBadRequest)
			return
		}

		// Validate message content
		messageContent := payload.Query
		if messageContent == "" {
			messageContent = payload.Message
		}
		if messageContent == "" {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   "Missing message content: either query or message field is required",
			})
			auth.SendAuthErrorResponse(w, "Either query or message field is required", http.StatusBadRequest)
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
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error creating forward message: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Error creating forward message", http.StatusInternalServerError)
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
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error creating message content: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Error creating message content", http.StatusInternalServerError)
			return
		}

		// Create a websocket message - using token owner as recipient
		recipientID := fromUserID // Use authenticated user as recipient
		wsMessage := models.Message{
			From:             fromUserID,
			To:               recipientID, // Set recipient to token owner's ID
			Timestamp:        time.Now(),
			Status:           "pending",
			Content:          string(content),
			IsForwardMessage: true,
		}

		// Log the message for security auditing
		log.Printf("Processing direct message from %s to %s", fromUserID, recipientID)
		securityLogger.LogAuthEvent(auth.SecurityEvent{
			Timestamp: time.Now(),
			Event:     auth.EventDirectMessageSending,
			UserID:    fromUserID,
			IP:        clientIP,
			Success:   true,
			Details:   fmt.Sprintf("Sending message to recipient: %s", recipientID),
		})

		// Register a response channel for the recipient
		responseCh := wsServer.RegisterResponseChannel(recipientID)
		defer wsServer.RemoveResponseChannel(recipientID)

		// Set up a timeout context
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Try to deliver the message via the websocket server
		if err := wsServer.DeliverHTTPMessage(wsMessage); err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error delivering message to %s: %v", recipientID, err),
			})
			log.Printf("Error delivering message to %s: %v", recipientID, err)
			auth.SendAuthErrorResponse(w, "Error delivering message", http.StatusInternalServerError)
			return
		}

		// Wait for response with timeout
		var responseMsg models.Message
		select {
		case responseMsg = <-responseCh:
			// Received response - verify it's actually from the recipient
			if responseMsg.From != recipientID {
				securityLogger.LogAuthEvent(auth.SecurityEvent{
					Timestamp: time.Now(),
					Event:     auth.EventDirectMessageSending,
					UserID:    fromUserID,
					IP:        clientIP,
					Success:   false,
					Details: fmt.Sprintf("Security alert: Received response from %s but expected from %s",
						responseMsg.From, recipientID),
				})
				log.Printf("Security alert: Received response from %s but expected from %s",
					responseMsg.From, recipientID)
				auth.SendAuthErrorResponse(w, "Invalid response source", http.StatusInternalServerError)
				return
			}
			log.Printf("Received response from recipient %s", recipientID)
		case <-ctx.Done():
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Request timed out waiting for response from %s", recipientID),
			})
			auth.SendAuthErrorResponse(w, "Request timed out waiting for response", http.StatusGatewayTimeout)
			return
		}

		// Parse the response
		var responseWrapper struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(responseMsg.Content), &responseWrapper); err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error parsing response: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Error parsing response", http.StatusInternalServerError)
			return
		}

		var forwardResponse struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal([]byte(responseWrapper.Message), &forwardResponse); err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error parsing forward response: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Error parsing forward response", http.StatusInternalServerError)
			return
		}

		// Log successful message delivery
		securityLogger.LogAuthEvent(auth.SecurityEvent{
			Timestamp: time.Now(),
			Event:     auth.EventDirectMessageSending,
			UserID:    fromUserID,
			IP:        clientIP,
			Success:   true,
			Details:   fmt.Sprintf("Successfully delivered message to %s and received response", recipientID),
		})

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
