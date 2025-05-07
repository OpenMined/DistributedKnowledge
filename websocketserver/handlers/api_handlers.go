package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"websocketserver/auth"
	"websocketserver/models"
)

// HandleUserAPIs returns an HTTP handler that allows authenticated users to manage their APIs.
// This handler supports both GET (to retrieve APIs) and POST (to create/update APIs).
func HandleUserAPIs(authService *auth.Service, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		switch r.Method {
		case http.MethodGet:
			// Retrieve user's APIs
			query := "SELECT id, api_name, documents, policy, created_at, updated_at FROM user_apis WHERE user_id = ?"
			rows, err := db.Query(query, userID)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var apis []models.API
			for rows.Next() {
				var api models.API
				var documentsJSON string
				var createdAt, updatedAt time.Time

				err := rows.Scan(
					&api.ID,
					&api.APIName,
					&documentsJSON,
					&api.Policy,
					&createdAt,
					&updatedAt,
				)
				if err != nil {
					http.Error(w, "Error parsing API data", http.StatusInternalServerError)
					return
				}

				// Parse documents JSON array
				if documentsJSON != "" {
					if err := json.Unmarshal([]byte(documentsJSON), &api.Documents); err != nil {
						http.Error(w, "Error parsing API documents", http.StatusInternalServerError)
						return
					}
				}

				api.UserID = userID
				api.CreatedAt = createdAt
				api.UpdatedAt = updatedAt
				apis = append(apis, api)
			}

			if err = rows.Err(); err != nil {
				http.Error(w, "Error iterating API rows", http.StatusInternalServerError)
				return
			}

			// If no APIs found, return an empty array
			if apis == nil {
				apis = []models.API{}
			}

			// Return APIs as JSON
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(apis); err != nil {
				http.Error(w, "Error encoding API data", http.StatusInternalServerError)
				return
			}

		case http.MethodPost:
			// Create or update an API
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			var api models.API
			if err := json.Unmarshal(body, &api); err != nil {
				http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
				return
			}

			// Validate required fields
			if api.APIName == "" {
				http.Error(w, "API name is required", http.StatusBadRequest)
				return
			}

			// Set user ID from token
			api.UserID = userID

			// Convert documents to JSON
			var documentsJSON []byte
			if len(api.Documents) > 0 {
				documentsJSON, err = json.Marshal(api.Documents)
				if err != nil {
					http.Error(w, "Error encoding API documents", http.StatusInternalServerError)
					return
				}
			}

			// Begin transaction
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

			// Check if API exists for this user
			var existingID int
			query := "SELECT id FROM user_apis WHERE user_id = ? AND api_name = ?"
			err = tx.QueryRow(query, userID, api.APIName).Scan(&existingID)

			now := time.Now()
			if err == sql.ErrNoRows {
				// Insert new API
				insertQuery := `
					INSERT INTO user_apis
					(user_id, api_name, documents, policy, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?)
				`
				_, err = tx.Exec(
					insertQuery,
					userID,
					api.APIName,
					string(documentsJSON),
					api.Policy,
					now,
					now,
				)
				if err != nil {
					http.Error(w, "Database error inserting API", http.StatusInternalServerError)
					return
				}
			} else if err == nil {
				// Update existing API
				updateQuery := `
					UPDATE user_apis
					SET documents = ?, policy = ?, updated_at = ?
					WHERE id = ?
				`
				_, err = tx.Exec(
					updateQuery,
					string(documentsJSON),
					api.Policy,
					now,
					existingID,
				)
				if err != nil {
					http.Error(w, "Database error updating API", http.StatusInternalServerError)
					return
				}
			} else {
				// Other database error
				http.Error(w, "Database error checking for existing API", http.StatusInternalServerError)
				return
			}

			// Commit transaction
			if err = tx.Commit(); err != nil {
				http.Error(w, "Database commit error", http.StatusInternalServerError)
				return
			}
			commit = true

			// Return success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API saved successfully"))

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleGetPublicAPIs returns an HTTP handler that allows retrieving all APIs.
func HandleGetPublicAPIs(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Query all APIs
		query := `
			SELECT a.id, a.user_id, u.username, a.api_name, a.documents, a.policy, a.created_at, a.updated_at
			FROM user_apis a
			JOIN users u ON a.user_id = u.user_id
			ORDER BY a.updated_at DESC
		`
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PublicAPI struct {
			models.API
			Username string `json:"username"`
		}

		var apis []PublicAPI
		for rows.Next() {
			var api PublicAPI
			var documentsJSON string
			var createdAt, updatedAt time.Time

			err := rows.Scan(
				&api.ID,
				&api.UserID,
				&api.Username,
				&api.APIName,
				&documentsJSON,
				&api.Policy,
				&createdAt,
				&updatedAt,
			)
			if err != nil {
				http.Error(w, "Error parsing API data", http.StatusInternalServerError)
				return
			}

			// Parse documents JSON array
			if documentsJSON != "" {
				if err := json.Unmarshal([]byte(documentsJSON), &api.Documents); err != nil {
					http.Error(w, "Error parsing API documents", http.StatusInternalServerError)
					return
				}
			}

			api.CreatedAt = createdAt
			api.UpdatedAt = updatedAt
			apis = append(apis, api)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, "Error iterating API rows", http.StatusInternalServerError)
			return
		}

		// If no APIs found, return an empty array
		if apis == nil {
			apis = []PublicAPI{}
		}

		// Return APIs as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(apis); err != nil {
			http.Error(w, "Error encoding API data", http.StatusInternalServerError)
			return
		}
	}
}
