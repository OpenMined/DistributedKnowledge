package http

import (
	"context"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AnswerResponse represents the response structure for GET /answers endpoint
type AnswerResponse struct {
	Query   string            `json:"query"`
	Answers map[string]string `json:"answers"`
}

// QueryRequest represents the request structure for the /answers endpoint
type QueryRequest struct {
	Query string `json:"query"`
}

// HandleGetAnswersByQuery handles the /answers endpoint
// It can accept the query either as a URL parameter or as a JSON body
func HandleGetAnswersByQuery(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Println("============ HandleGetAnswersByQuery CALLED ============")

	// First check if this is a POST with JSON body
	var query string
	contentType := r.Header.Get("Content-Type")

	if r.Method == "POST" && strings.Contains(contentType, "application/json") {
		// Parse JSON body
		var req QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("[ERROR] Failed to parse JSON body: %v\n", err)
			sendErrorResponse(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}
		query = req.Query
		fmt.Printf("[DEBUG] Received query from JSON body: '%s'\n", query)
	} else {
		// Fall back to URL query parameter
		query = r.URL.Query().Get("query")
		fmt.Printf("[DEBUG] Received query parameter: '%s'\n", query)
	}

	fmt.Printf("[DEBUG] Request Headers: %v\n", r.Header)
	fmt.Printf("[DEBUG] Request URL: %s\n", r.URL.String())

	if strings.TrimSpace(query) == "" {
		fmt.Println("[ERROR] Empty query parameter received")
		sendErrorResponse(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	fmt.Println("[DEBUG] Attempting to retrieve database connection from context")
	dbInstance, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		fmt.Printf("[ERROR] Failed to retrieve database instance: %v\n", err)
		sendErrorResponse(w, fmt.Sprintf("Couldn't retrieve database instance: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Println("[DEBUG] Successfully retrieved database connection")

	// Get answers for the query
	fmt.Printf("[DEBUG] Executing SQL query for answers with query string: '%s'\n", query)
	answers, err := db.AnswersForQuestion(ctx, dbInstance, query)
	if err != nil {
		fmt.Printf("[ERROR] Database error when retrieving answers: %v\n", err)
		sendErrorResponse(w, fmt.Sprintf("Error retrieving answers for query '%s': %v", query, err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG] Query executed successfully, found %d answers\n", len(answers))

	// Check if answers were found
	if len(answers) == 0 {
		fmt.Printf("[DEBUG] No answers found for query: '%s'\n", query)
		sendErrorResponse(w, fmt.Sprintf("No answers found for query: '%s'", query), http.StatusNotFound)
		return
	}

	// Log found answers
	fmt.Println("[DEBUG] Answers found:")
	for user, answer := range answers {
		fmt.Printf("  - User '%s': Answer text (length: %d): '%s'\n", user, len(answer), answer)
	}

	// Prepare response
	response := AnswerResponse{
		Query:   query,
		Answers: answers,
	}

	fmt.Printf("[DEBUG] Preparing response JSON with %d answers\n", len(answers))
	responseJSON, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("[ERROR] Failed to marshal response to JSON: %v\n", err)
		sendErrorResponse(w, "Internal server error marshaling response", http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG] Response JSON size: %d bytes\n", len(responseJSON))

	// Send successful response
	fmt.Println("[DEBUG] Sending successful response")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		fmt.Printf("[ERROR] Failed to write response: %v\n", err)
		return
	}

	fmt.Println("============ HandleGetAnswersByQuery COMPLETED ============")
}
