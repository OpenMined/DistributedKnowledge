package http

import (
	"context"
	dk_client "dk/client"
	"dk/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RemoteMessageRequest represents the request body for sending remote messages
type RemoteMessageRequest struct {
	Question string   `json:"question"`
	Peers    []string `json:"peers"`
}

// HandleSendRemoteMessage processes HTTP POST requests for sending remote messages to peers
func HandleSendRemoteMessage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req RemoteMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the question field
	if req.Question == "" {
		sendErrorResponse(w, "Question parameter is required", http.StatusBadRequest)
		return
	}

	// Get the DK client from context
	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve DK client from context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the remote message
	query := utils.RemoteMessage{
		Type:    "query",
		Message: req.Question,
	}

	// Marshal the query to JSON
	jsonData, err := json.Marshal(query)
	if err != nil {
		sendErrorResponse(w, "Failed to marshal query: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send the message based on whether peers were specified
	if len(req.Peers) == 0 {
		// Broadcast message to all peers
		err = dkClient.BroadcastMessage(string(jsonData))
		if err != nil {
			sendErrorResponse(w, "Failed to broadcast message: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Send message to specific peers
		for _, peer := range req.Peers {
			err = dkClient.SendMessage(dk_client.Message{
				From:      dkClient.UserID,
				To:        peer,
				Content:   string(jsonData),
				Timestamp: time.Now(),
			})
			if err != nil {
				sendErrorResponse(w, fmt.Sprintf("Failed to send message to peer %s: %s", peer, err.Error()), http.StatusInternalServerError)
				return
			}
		}
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Query '%s' sent successfully", req.Question),
	})
}
