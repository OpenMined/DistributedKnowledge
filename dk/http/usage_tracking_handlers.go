package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"dk/db"
)

// UsageResponse represents the API usage response data
type UsageResponse struct {
	Items []*db.APIUsage `json:"items"`
	Total int            `json:"total"`
}

// UsageSummaryResponse represents the API usage summary response data
type UsageSummaryResponse struct {
	Items []*db.APIUsageSummary `json:"items"`
}

// NotificationResponse represents the notification response data
type NotificationResponse struct {
	Items []*db.QuotaNotification `json:"items"`
	Total int                     `json:"total"`
}

// RegisterUsageTrackingHandlers registers usage tracking HTTP endpoints
func RegisterUsageTrackingHandlers(r *mux.Router, dbConn *db.DatabaseConnection) {
	// Usage data endpoints
	r.HandleFunc("/api/v1/usage", handleGetAllUsage(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/usage/{apiId}", handleGetAPIUsage(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/usage/{apiId}/user/{userId}", handleGetUserAPIUsage(dbConn)).Methods("GET")

	// Usage summary endpoints
	r.HandleFunc("/api/v1/usage-summary", handleGetAllUsageSummaries(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/usage-summary/{apiId}", handleGetAPISummaries(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/usage-summary/{apiId}/user/{userId}", handleGetUserAPISummaries(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/usage-summary/refresh", handleRefreshUsageSummaries(dbConn)).Methods("POST")

	// Notification endpoints
	r.HandleFunc("/api/v1/notifications", handleGetAllNotifications(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/notifications/user/{userId}", handleGetUserNotifications(dbConn)).Methods("GET")
	r.HandleFunc("/api/v1/notifications/{id}/read", handleMarkNotificationAsRead(dbConn)).Methods("PUT")
	r.HandleFunc("/api/v1/notifications/{id}", handleDeleteNotification(dbConn)).Methods("DELETE")
	r.HandleFunc("/api/v1/notifications/cleanup", handleCleanupNotifications(dbConn)).Methods("POST")
}

// handleGetAllUsage handles retrieving usage data across all APIs
func handleGetAllUsage(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		fromDateStr := r.URL.Query().Get("from")
		toDateStr := r.URL.Query().Get("to")
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		var fromDate, toDate time.Time
		var err error

		// Parse date parameters
		if fromDateStr != "" {
			fromDate, err = time.Parse(time.RFC3339, fromDateStr)
			if err != nil {
				http.Error(w, "Invalid from date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		if toDateStr != "" {
			toDate, err = time.Parse(time.RFC3339, toDateStr)
			if err != nil {
				http.Error(w, "Invalid to date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		// Parse pagination parameters
		limit := 50 // Default
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}

		offset := 0 // Default
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil || offset < 0 {
				http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
				return
			}
		}

		// Get usage data
		items, _, err := db.GetAllAPIUsage(dbConn.DB, fromDate, toDate, limit, offset)
		if err != nil {
			http.Error(w, "Failed to get usage data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		response := UsageResponse{
			Items: items,
			Total: len(items),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleGetAPIUsage handles retrieving usage data for a specific API
func handleGetAPIUsage(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API ID from path
		vars := mux.Vars(r)
		apiID := vars["apiId"]

		// Parse query parameters
		fromDateStr := r.URL.Query().Get("from")
		toDateStr := r.URL.Query().Get("to")
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		var fromDate, toDate time.Time
		var err error

		// Parse date parameters
		if fromDateStr != "" {
			fromDate, err = time.Parse(time.RFC3339, fromDateStr)
			if err != nil {
				http.Error(w, "Invalid from date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		if toDateStr != "" {
			toDate, err = time.Parse(time.RFC3339, toDateStr)
			if err != nil {
				http.Error(w, "Invalid to date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		// Parse pagination parameters
		limit := 50 // Default
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}

		offset := 0 // Default
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil || offset < 0 {
				http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
				return
			}
		}

		// Verify API exists
		api, err := db.GetAPI(dbConn.DB, apiID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "API not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to verify API: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Get usage data - we'll need to adapt our query for this specific case
		// This is a simplification - in a real implementation, we'd need to query all users for this API
		// For now, let's just get all usage and filter by API ID
		items, _, err := db.GetAllAPIUsage(dbConn.DB, fromDate, toDate, limit, offset)
		if err != nil {
			http.Error(w, "Failed to get usage data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Filter by API ID
		var filteredItems []*db.APIUsage
		for _, item := range items {
			if item.APIID == api.ID {
				filteredItems = append(filteredItems, item)
			}
		}

		// Respond with JSON
		response := UsageResponse{
			Items: filteredItems,
			Total: len(filteredItems), // This is simplified; in a real implementation, we'd query for total
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleGetUserAPIUsage handles retrieving usage data for a specific user and API
func handleGetUserAPIUsage(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API ID and user ID from path
		vars := mux.Vars(r)
		apiID := vars["apiId"]
		userID := vars["userId"]

		// Parse query parameters
		fromDateStr := r.URL.Query().Get("from")
		toDateStr := r.URL.Query().Get("to")
		limitStr := r.URL.Query().Get("limit")

		var fromDate, toDate time.Time
		var err error

		// Parse date parameters
		if fromDateStr != "" {
			fromDate, err = time.Parse(time.RFC3339, fromDateStr)
			if err != nil {
				http.Error(w, "Invalid from date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		if toDateStr != "" {
			toDate, err = time.Parse(time.RFC3339, toDateStr)
			if err != nil {
				http.Error(w, "Invalid to date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		} else {
			toDate = time.Now() // Default to now if not specified
		}

		// Parse limit parameter
		limit := 50 // Default
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}

		// Verify API exists
		_, err = db.GetAPI(dbConn.DB, apiID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "API not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to verify API: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Get usage data for the specific period if dates provided, otherwise recent usage
		var items []*db.APIUsage
		var total int

		if !fromDate.IsZero() && !toDate.IsZero() {
			items, err = db.GetUsageByPeriod(dbConn.DB, apiID, userID, fromDate, toDate)
			total = len(items)
		} else {
			items, err = db.GetRecentAPIUsage(dbConn.DB, apiID, userID, limit)
			total = len(items)
		}

		if err != nil {
			http.Error(w, "Failed to get usage data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		response := UsageResponse{
			Items: items,
			Total: total,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleGetAllUsageSummaries handles retrieving usage summaries
func handleGetAllUsageSummaries(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This endpoint would need to gather summaries across all APIs
		// For now, returning an empty response with 501 Not Implemented
		// In a real implementation, we'd query summaries across all APIs

		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// handleGetAPISummaries handles retrieving usage summaries for a specific API
func handleGetAPISummaries(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API ID from path
		vars := mux.Vars(r)
		apiID := vars["apiId"]

		// Parse query parameters
		periodType := r.URL.Query().Get("period") // daily, weekly, monthly
		fromDateStr := r.URL.Query().Get("from")
		toDateStr := r.URL.Query().Get("to")

		var fromDate, toDate time.Time
		var err error

		// Parse date parameters
		if fromDateStr != "" {
			fromDate, err = time.Parse(time.RFC3339, fromDateStr)
			if err != nil {
				http.Error(w, "Invalid from date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		if toDateStr != "" {
			toDate, err = time.Parse(time.RFC3339, toDateStr)
			if err != nil {
				http.Error(w, "Invalid to date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		// Verify API exists
		_, err = db.GetAPI(dbConn.DB, apiID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "API not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to verify API: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Get summaries for the API
		summaries, err := db.GetAPIUsageSummaries(dbConn.DB, apiID, "", periodType, fromDate, toDate)
		if err != nil {
			http.Error(w, "Failed to get usage summaries: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		response := UsageSummaryResponse{
			Items: summaries,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleGetUserAPISummaries handles retrieving usage summaries for a specific user and API
func handleGetUserAPISummaries(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API ID and user ID from path
		vars := mux.Vars(r)
		apiID := vars["apiId"]
		userID := vars["userId"]

		// Parse query parameters
		periodType := r.URL.Query().Get("period") // daily, weekly, monthly
		fromDateStr := r.URL.Query().Get("from")
		toDateStr := r.URL.Query().Get("to")

		var fromDate, toDate time.Time
		var err error

		// Parse date parameters
		if fromDateStr != "" {
			fromDate, err = time.Parse(time.RFC3339, fromDateStr)
			if err != nil {
				http.Error(w, "Invalid from date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		if toDateStr != "" {
			toDate, err = time.Parse(time.RFC3339, toDateStr)
			if err != nil {
				http.Error(w, "Invalid to date format. Use RFC3339", http.StatusBadRequest)
				return
			}
		}

		// Verify API exists
		_, err = db.GetAPI(dbConn.DB, apiID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "API not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to verify API: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Get summaries for the specific API and user
		summaries, err := db.GetAPIUsageSummaries(dbConn.DB, apiID, userID, periodType, fromDate, toDate)
		if err != nil {
			http.Error(w, "Failed to get usage summaries: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		response := UsageSummaryResponse{
			Items: summaries,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleRefreshUsageSummaries handles triggering a refresh of all usage summaries
func handleRefreshUsageSummaries(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.UpdateAPIUsageSummaries(dbConn.DB)
		if err != nil {
			http.Error(w, "Failed to refresh usage summaries: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Usage summaries refreshed"})
	}
}

// handleGetAllNotifications handles retrieving all notifications
func handleGetAllNotifications(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This endpoint would need to gather notifications across all APIs and users
		// For now, returning an empty response with 501 Not Implemented
		// In a real implementation, we'd query notifications with proper pagination

		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// handleGetUserNotifications handles retrieving notifications for a specific user
func handleGetUserNotifications(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from path
		vars := mux.Vars(r)
		userID := vars["userId"]

		// Parse query parameters
		unreadOnlyStr := r.URL.Query().Get("unread_only")
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		// Parse boolean parameter
		unreadOnly := false
		if unreadOnlyStr == "true" {
			unreadOnly = true
		}

		var err error

		// Parse pagination parameters
		limit := 50 // Default
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}

		offset := 0 // Default
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil || offset < 0 {
				http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
				return
			}
		}

		// Get notifications for the user
		notifications, total, err := db.GetUserNotifications(dbConn.DB, userID, unreadOnly, limit, offset)
		if err != nil {
			http.Error(w, "Failed to get notifications: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		response := NotificationResponse{
			Items: notifications,
			Total: total,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handleMarkNotificationAsRead handles marking a notification as read
func handleMarkNotificationAsRead(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get notification ID from path
		vars := mux.Vars(r)
		notificationID := vars["id"]

		// Mark as read
		err := db.MarkNotificationAsRead(dbConn.DB, notificationID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "Notification not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to mark notification as read: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Notification marked as read"})
	}
}

// handleDeleteNotification handles deleting a notification
func handleDeleteNotification(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get notification ID from path
		vars := mux.Vars(r)
		notificationID := vars["id"]

		// Delete the notification
		err := db.DeleteQuotaNotification(dbConn.DB, notificationID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "Notification not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to delete notification: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Return success
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCleanupNotifications handles cleaning up old read notifications
func handleCleanupNotifications(dbConn *db.DatabaseConnection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to get age parameter
		var request struct {
			AgeDays int `json:"age_days"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if request.AgeDays <= 0 {
			http.Error(w, "Age days must be a positive integer", http.StatusBadRequest)
			return
		}

		// Calculate age duration
		age := time.Duration(request.AgeDays) * 24 * time.Hour

		// Delete old read notifications
		count, err := db.DeleteAllReadNotifications(dbConn.DB, age)
		if err != nil {
			http.Error(w, "Failed to clean up notifications: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return success with count
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Old notifications cleaned up",
			"count":   count,
		})
	}
}
