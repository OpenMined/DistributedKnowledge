package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DirectMessagePayload represents the JSON payload for direct messages via HTTP
type DirectMessagePayload struct {
	Type     string `json:"type"`
	Query    string `json:"query,omitempty"`
	Message  string `json:"message,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
	// Recipient field removed - server now uses token owner's ID as recipient
}

// DirectMessageResponse represents the expected response from the direct message API
type DirectMessageResponse struct {
	Success bool   `json:"success"`
	Answer  string `json:"answer"`
}

// SendDirectMessage sends a query via the Direct Message API
// Note: This function now only sends messages to the authenticated user (token owner)
// as the server no longer accepts a recipient field
func (c *Client) SendDirectMessage(queryText string, _ string) (string, error) {
	if c.jwtToken == "" {
		return "", fmt.Errorf("JWT token is not set; please login first")
	}

	// Construct the endpoint URL
	endpoint := fmt.Sprintf("%s/direct-message/", c.serverURL)

	// Create the payload without recipient field
	payload := DirectMessagePayload{
		Type:  "forward",
		Query: queryText,
	}

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.jwtToken)

	// Send the request
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send direct message: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("direct message request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var response DirectMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Return the answer string
	return response.Answer, nil
}

// QuerySelf sends a query to the current user's own DK instance
// This is now the preferred method for sending direct messages, as all messages
// are automatically routed to the authenticated user
func (c *Client) QuerySelf(queryText string) (string, error) {
	return c.SendDirectMessage(queryText, "")
}

// RegisterDocument registers a new document with the RAG system
func (c *Client) RegisterDocument(filename string, content string) (string, error) {
	if c.jwtToken == "" {
		return "", fmt.Errorf("JWT token is not set; please login first")
	}

	// Construct the endpoint URL - use the register-document endpoint
	endpoint := fmt.Sprintf("%s/register-document/", c.serverURL)

	// Create the payload for document registration
	payload := DirectMessagePayload{
		Type:     "forward", // Keep 'forward' type as per spec
		Filename: filename,
		Content:  content,
	}

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.jwtToken)

	// Send the request
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to register document: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("document registration request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response - match the structure of the response from the server
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Return the message string
	return response.Message, nil
}
