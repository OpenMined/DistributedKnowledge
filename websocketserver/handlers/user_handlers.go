package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"websocketserver/auth"
)

// HandleGetUserDescriptions returns an HTTP GET endpoint that returns the list of descriptions
// for a specified user. The user id is provided as part of the URL path like /user/descriptions/<user_id>.
// No authentication is required.
func HandleGetUserDescriptions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests.
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Expecting the URL to be: /user/descriptions/<user_id>
		// Split the URL path into components.
		// For example, if the URL is "/user/descriptions/someUserID",
		// parts[0] is empty, parts[1] is "user", parts[2] is "descriptions",
		// and parts[3] is the user id.
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, "User ID not specified in URL", http.StatusBadRequest)
			return
		}
		userID := parts[3]

		// Query the database for the descriptions JSON string.
		var storedDescriptions string
		query := "SELECT descriptions FROM user_descriptions WHERE user_id = ?"
		if err := db.QueryRow(query, userID).Scan(&storedDescriptions); err != nil {
			if err == sql.ErrNoRows {
				// If no record exists for this user, return an empty JSON array.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("[]"))
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Return the retrieved descriptions. Since storedDescriptions is already a JSON array,
		// we simply send it back.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(storedDescriptions))
	}
}

// HandleUserDescriptions returns an HTTP handler that allows authenticated users to set
// their descriptions list by sending a JSON array of strings. This request replaces any previously stored list.
func HandleUserDescriptions(authService *auth.Service, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow only POST requests.
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract and validate the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]

		// Validate the token and get the user ID.
		claims, err := auth.ParseToken(tokenStr, authService)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Read and parse the JSON payload into a slice of strings.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		var newDescriptions []string
		if err := json.Unmarshal(body, &newDescriptions); err != nil {
			http.Error(w, "Invalid JSON payload, expected an array of strings", http.StatusBadRequest)
			return
		}

		// Optionally, ensure the array is valid (for example, not nil)
		if len(newDescriptions) == 0 {
			http.Error(w, "Descriptions list cannot be empty", http.StatusBadRequest)
			return
		}

		// Marshal the new list to JSON for storage.
		updatedList, err := json.Marshal(newDescriptions)
		if err != nil {
			http.Error(w, "Error processing descriptions list", http.StatusInternalServerError)
			return
		}

		// Begin a transaction for atomic update.
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		commit := false
		defer func() {
			if !commit {
				tx.Rollback()
			}
		}()

		// Check for an existing record for the user.
		var existing string
		query := "SELECT descriptions FROM user_descriptions WHERE user_id = ?"
		err = tx.QueryRow(query, userID).Scan(&existing)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err == sql.ErrNoRows {
			// No record exists; insert a new one.
			insertQuery := "INSERT INTO user_descriptions (user_id, descriptions) VALUES (?, ?)"
			if _, err := tx.Exec(insertQuery, userID, string(updatedList)); err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		} else {
			// Record exists; replace the list.
			updateQuery := "UPDATE user_descriptions SET descriptions = ? WHERE user_id = ?"
			if _, err = tx.Exec(updateQuery, string(updatedList), userID); err != nil {
				http.Error(w, "Database error updating descriptions", http.StatusInternalServerError)
				return
			}
		}

		// Commit the transaction.
		if err = tx.Commit(); err != nil {
			http.Error(w, "Database commit error", http.StatusInternalServerError)
			return
		}
		commit = true

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Descriptions list updated"))
	}
}
