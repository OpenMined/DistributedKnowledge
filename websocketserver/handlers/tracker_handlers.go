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

// HandleUserTrackers returns an HTTP handler that allows authenticated users to manage their trackers.
// This handler supports both GET (to retrieve trackers) and POST (to create/update trackers).
func HandleUserTrackers(authService *auth.Service, db *sql.DB) http.HandlerFunc {
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
			// Retrieve user's trackers
			query := "SELECT id, tracker_name, tracker_description, tracker_version, tracker_documents, created_at, updated_at FROM user_trackers WHERE user_id = ?"
			rows, err := db.Query(query, userID)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var trackers []models.Tracker
			for rows.Next() {
				var tracker models.Tracker
				var trackerDocumentsJSON string
				var createdAt, updatedAt time.Time

				err := rows.Scan(
					&tracker.ID,
					&tracker.TrackerName,
					&tracker.TrackerDescription,
					&tracker.TrackerVersion,
					&trackerDocumentsJSON,
					&createdAt,
					&updatedAt,
				)
				if err != nil {
					http.Error(w, "Error parsing tracker data", http.StatusInternalServerError)
					return
				}

				// Parse tracker_documents JSON object (with datasets and templates)
				if trackerDocumentsJSON != "" {
					if err := json.Unmarshal([]byte(trackerDocumentsJSON), &tracker.TrackerDocuments); err != nil {
						http.Error(w, "Error parsing tracker documents", http.StatusInternalServerError)
						return
					}
				}

				tracker.UserID = userID
				tracker.CreatedAt = createdAt
				tracker.UpdatedAt = updatedAt
				trackers = append(trackers, tracker)
			}

			if err = rows.Err(); err != nil {
				http.Error(w, "Error iterating tracker rows", http.StatusInternalServerError)
				return
			}

			// If no trackers found, return an empty array
			if trackers == nil {
				trackers = []models.Tracker{}
			}

			// Return trackers as JSON
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(trackers); err != nil {
				http.Error(w, "Error encoding tracker data", http.StatusInternalServerError)
				return
			}

		case http.MethodPost:
			// Update the entire list of trackers for a user
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			// Parse TrackerListPayload format
			var trackerList models.TrackerListPayload
			if err := json.Unmarshal(body, &trackerList); err != nil {
				http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
				return
			}
			
			if len(trackerList.Trackers) == 0 {
				http.Error(w, "Tracker list cannot be empty", http.StatusBadRequest)
				return
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

			// 1. First, get all existing trackers for this user to identify which ones to delete
			existingTrackersQuery := "SELECT tracker_name FROM user_trackers WHERE user_id = ?"
			rows, err := tx.Query(existingTrackersQuery, userID)
			if err != nil {
				http.Error(w, "Database error fetching existing trackers", http.StatusInternalServerError)
				return
			}
			
			existingTrackers := make(map[string]bool)
			for rows.Next() {
				var trackerName string
				if err := rows.Scan(&trackerName); err != nil {
					rows.Close()
					http.Error(w, "Error scanning tracker data", http.StatusInternalServerError)
					return
				}
				existingTrackers[trackerName] = true
			}
			rows.Close()
			
			if err = rows.Err(); err != nil {
				http.Error(w, "Error iterating tracker rows", http.StatusInternalServerError)
				return
			}
			
			// 2. Process each tracker in the new list - update or insert
			now := time.Now()
			for trackerName, trackerData := range trackerList.Trackers {
				// Convert tracker_documents to JSON
				var trackerDocumentsJSON []byte
				hasData := (len(trackerData.TrackerDocuments.Datasets) > 0 || len(trackerData.TrackerDocuments.Templates) > 0)
				if hasData {
					trackerDocumentsJSON, err = json.Marshal(trackerData.TrackerDocuments)
					if err != nil {
						http.Error(w, "Error encoding tracker documents", http.StatusInternalServerError)
						return
					}
				}
				
				if existing := existingTrackers[trackerName]; existing {
					// Update existing tracker
					updateQuery := `
						UPDATE user_trackers 
						SET tracker_description = ?, tracker_version = ?, tracker_documents = ?, updated_at = ?
						WHERE user_id = ? AND tracker_name = ?
					`
					_, err = tx.Exec(
						updateQuery,
						trackerData.TrackerDescription,
						trackerData.TrackerVersion,
						string(trackerDocumentsJSON),
						now,
						userID,
						trackerName,
					)
					if err != nil {
						http.Error(w, "Database error updating tracker: "+trackerName, http.StatusInternalServerError)
						return
					}
					
					// Remove from existingTrackers map to mark as processed
					delete(existingTrackers, trackerName)
				} else {
					// Insert new tracker
					insertQuery := `
						INSERT INTO user_trackers 
						(user_id, tracker_name, tracker_description, tracker_version, tracker_documents, created_at, updated_at) 
						VALUES (?, ?, ?, ?, ?, ?, ?)
					`
					_, err = tx.Exec(
						insertQuery,
						userID,
						trackerName,
						trackerData.TrackerDescription,
						trackerData.TrackerVersion,
						string(trackerDocumentsJSON),
						now,
						now,
					)
					if err != nil {
						http.Error(w, "Database error inserting tracker: "+trackerName, http.StatusInternalServerError)
						return
					}
				}
			}
			
			// 3. Delete any trackers that were not in the updated list
			if len(existingTrackers) > 0 {
				for trackerName := range existingTrackers {
					deleteQuery := "DELETE FROM user_trackers WHERE user_id = ? AND tracker_name = ?"
					_, err = tx.Exec(deleteQuery, userID, trackerName)
					if err != nil {
						http.Error(w, "Database error deleting tracker: "+trackerName, http.StatusInternalServerError)
						return
					}
				}
			}
			
			// Commit transaction
			if err = tx.Commit(); err != nil {
				http.Error(w, "Database commit error", http.StatusInternalServerError)
				return
			}
			commit = true
			
			// Return success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Tracker list updated successfully"))

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleGetPublicTrackers returns an HTTP handler that allows retrieving all trackers.
func HandleGetPublicTrackers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Query all trackers
		query := `
			SELECT t.id, t.user_id, u.username, t.tracker_name, t.tracker_description, 
			       t.tracker_version, t.tracker_documents, t.created_at, t.updated_at 
			FROM user_trackers t
			JOIN users u ON t.user_id = u.user_id
			ORDER BY t.updated_at DESC
		`
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PublicTracker struct {
			models.Tracker
			Username string `json:"username"`
		}

		var trackers []PublicTracker
		for rows.Next() {
			var tracker PublicTracker
			var trackerDocumentsJSON string
			var createdAt, updatedAt time.Time

			err := rows.Scan(
				&tracker.ID,
				&tracker.UserID,
				&tracker.Username,
				&tracker.TrackerName,
				&tracker.TrackerDescription,
				&tracker.TrackerVersion,
				&trackerDocumentsJSON,
				&createdAt,
				&updatedAt,
			)
			if err != nil {
				http.Error(w, "Error parsing tracker data", http.StatusInternalServerError)
				return
			}

			// Parse tracker_documents JSON object (with datasets and templates)
			if trackerDocumentsJSON != "" {
				if err := json.Unmarshal([]byte(trackerDocumentsJSON), &tracker.TrackerDocuments); err != nil {
					http.Error(w, "Error parsing tracker documents: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

			tracker.CreatedAt = createdAt
			tracker.UpdatedAt = updatedAt
			trackers = append(trackers, tracker)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, "Error iterating tracker rows", http.StatusInternalServerError)
			return
		}

		// If no trackers found, return an empty array
		if trackers == nil {
			trackers = []PublicTracker{}
		}

		// Return trackers as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(trackers); err != nil {
			http.Error(w, "Error encoding tracker data", http.StatusInternalServerError)
			return
		}
	}
}