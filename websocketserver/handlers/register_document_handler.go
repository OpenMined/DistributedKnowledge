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

// RegisterDocumentPayload represents the JSON payload for document registration via HTTP
type RegisterDocumentPayload struct {
	Type     string `json:"type"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// HandleRegisterDocument handles POST requests to register documents via websocket
func HandleRegisterDocument(authService *auth.Service, wsServer *ws.Server) http.HandlerFunc {
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

		var payload RegisterDocumentPayload
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

		// Validate required fields
		if strings.TrimSpace(payload.Filename) == "" || strings.TrimSpace(payload.Content) == "" {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   "Missing required fields: filename and content are required",
			})
			auth.SendAuthErrorResponse(w, "Filename and content are required", http.StatusBadRequest)
			return
		}

		// Create the document registration message
		documentMsg := struct {
			Type     string `json:"type"`
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}{
			Type:     models.MessageTypeRegisterDocument,
			Filename: payload.Filename,
			Content:  payload.Content,
		}

		// Marshal the document message
		documentMsgJSON, err := json.Marshal(documentMsg)
		if err != nil {
			securityLogger.LogAuthEvent(auth.SecurityEvent{
				Timestamp: time.Now(),
				Event:     auth.EventDirectMessageSending,
				UserID:    fromUserID,
				IP:        clientIP,
				Success:   false,
				Details:   fmt.Sprintf("Error creating document registration message: %v", err),
			})
			auth.SendAuthErrorResponse(w, "Error creating document registration message", http.StatusInternalServerError)
			return
		}

		// Create a wrapper message
		wrapperMsg := struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{
			Type:    models.MessageTypeForward,
			Message: string(documentMsgJSON),
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
			To:               recipientID,
			Timestamp:        time.Now(),
			Status:           "pending",
			Content:          string(content),
			IsForwardMessage: true,
		}

		// Log the message for security auditing
		log.Printf("Processing document registration from %s for file: %s", fromUserID, payload.Filename)
		securityLogger.LogAuthEvent(auth.SecurityEvent{
			Timestamp: time.Now(),
			Event:     auth.EventDirectMessageSending,
			UserID:    fromUserID,
			IP:        clientIP,
			Success:   true,
			Details:   fmt.Sprintf("Sending document registration message to recipient: %s", recipientID),
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
			Details:   fmt.Sprintf("Successfully registered document for %s and received response", recipientID),
		})

		// Document registration response
		response := struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
			Type    string `json:"type"`
		}{
			Success: forwardResponse.Type == models.MessageTypeRegisterDocSuccess,
			Message: forwardResponse.Message,
			Type:    forwardResponse.Type,
		}

		// Return response to the HTTP client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}