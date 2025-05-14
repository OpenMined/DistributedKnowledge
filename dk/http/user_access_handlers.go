package http

import (
	"context"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HandleGetAPIUsers handles GET /api/apis/:id/users
func HandleGetAPIUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID from path
	apiID := r.PathValue("id")
	// For tests, check URL path since PathValue may not work in tests
	if apiID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			apiID = parts[3]
		}
	}
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	activeOnly := true // default to only active users
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		if val, err := strconv.ParseBool(activeStr); err == nil {
			activeOnly = val
		}
	}

	// Parse pagination parameters
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	// Parse sorting parameters
	sort := "granted_at" // default
	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		if sortParam == "granted_at" || sortParam == "user_id" {
			sort = sortParam
		}
	}

	order := "desc" // default
	if orderParam := r.URL.Query().Get("order"); orderParam != "" {
		if orderParam == "asc" || orderParam == "desc" {
			order = orderParam
		}
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if user is authorized (host user)
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get user access records
	accessRecords, total, err := db.ListAPIUserAccess(database, apiID, activeOnly, limit, offset, sort, order)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve user access records: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	users := make([]APIUserAccess, 0, len(accessRecords))
	for _, record := range accessRecords {
		// In a real implementation, you would fetch user details from your user store
		// For now, use placeholder data
		userName := "User " + record.ExternalUserID // Placeholder
		avatar := string(record.ExternalUserID[0])
		if avatar == "" {
			avatar = "U"
		}

		userDetails := UserRef{
			ID:     record.ExternalUserID,
			Name:   userName,
			Avatar: avatar,
		}

		user := APIUserAccess{
			ID:          record.ID,
			APIID:       record.APIID,
			UserID:      record.ExternalUserID,
			UserDetails: userDetails,
			AccessLevel: record.AccessLevel,
			GrantedAt:   record.GrantedAt,
			GrantedBy:   record.GrantedBy,
			IsActive:    record.IsActive,
			RevokedAt:   record.RevokedAt,
		}

		users = append(users, user)
	}

	response := APIUserListResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Users:  users,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGrantAPIAccess handles POST /api/apis/:id/users
func HandleGrantAPIAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID from path
	apiID := r.PathValue("id")
	// For tests, check URL path since PathValue may not work in tests
	if apiID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			apiID = parts[3]
		}
	}
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	var req APIUserAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.UserID == "" {
		sendErrorResponse(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if req.AccessLevel != "read" && req.AccessLevel != "write" && req.AccessLevel != "admin" {
		sendErrorResponse(w, "Access level must be 'read', 'write', or 'admin'", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if user is authorized (host user)
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Check if user already has access
	existingAccess, err := db.GetAPIUserAccessByUserID(database, apiID, req.UserID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		sendErrorResponse(w, "Failed to check existing access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if existingAccess != nil {
		// If access record exists but is inactive, reactivate it
		if !existingAccess.IsActive {
			existingAccess.IsActive = true
			existingAccess.RevokedAt = nil
			existingAccess.AccessLevel = req.AccessLevel // Update access level too

			if err := db.UpdateAPIUserAccess(database, existingAccess); err != nil {
				sendErrorResponse(w, "Failed to reactivate user access: "+err.Error(), http.StatusInternalServerError)
				return
			}

			response := APIUserAccessResponse{
				ID:          existingAccess.ID,
				APIID:       existingAccess.APIID,
				UserID:      existingAccess.ExternalUserID,
				AccessLevel: existingAccess.AccessLevel,
				IsActive:    existingAccess.IsActive,
				RevokedAt:   existingAccess.RevokedAt,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // 200 for update
			json.NewEncoder(w).Encode(response)
			return
		}

		// If already active, return conflict error
		sendErrorResponse(w, "User already has access to this API", http.StatusConflict)
		return
	}

	// Create new access record
	access := &db.APIUserAccess{
		ID:             uuid.New().String(),
		APIID:          apiID,
		ExternalUserID: req.UserID,
		AccessLevel:    req.AccessLevel,
		GrantedAt:      time.Now(),
		GrantedBy:      currentUserID,
		IsActive:       true,
	}

	if err := db.CreateAPIUserAccess(database, access); err != nil {
		sendErrorResponse(w, "Failed to grant user access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIUserAccessResponse{
		ID:          access.ID,
		APIID:       access.APIID,
		UserID:      access.ExternalUserID,
		AccessLevel: access.AccessLevel,
		IsActive:    access.IsActive,
		RevokedAt:   access.RevokedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 for creation
	json.NewEncoder(w).Encode(response)
}

// HandleUpdateAPIUserAccess handles PATCH /api/apis/:id/users/:user_id
func HandleUpdateAPIUserAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID and user ID from path
	apiID := r.PathValue("id")
	// For tests, check URL path since PathValue may not work in tests
	if apiID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			apiID = parts[3]
		}
	}
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	userID := r.PathValue("user_id")
	// For tests, check URL path since PathValue may not work in tests
	if userID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 6 {
			userID = parts[5]
		}
	}
	if userID == "" {
		sendErrorResponse(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var req APIUserAccessUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.AccessLevel != "read" && req.AccessLevel != "write" && req.AccessLevel != "admin" {
		sendErrorResponse(w, "Access level must be 'read', 'write', or 'admin'", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if user is authorized (host user)
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get existing access record
	access, err := db.GetAPIUserAccessByUserID(database, apiID, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "User access record not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve user access: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check if access is active
	if !access.IsActive {
		sendErrorResponse(w, "Cannot update revoked access. Restore access first.", http.StatusBadRequest)
		return
	}

	// Update access level
	access.AccessLevel = req.AccessLevel

	if err := db.UpdateAPIUserAccess(database, access); err != nil {
		sendErrorResponse(w, "Failed to update user access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIUserAccessResponse{
		ID:          access.ID,
		APIID:       access.APIID,
		UserID:      access.ExternalUserID,
		AccessLevel: access.AccessLevel,
		IsActive:    access.IsActive,
		RevokedAt:   access.RevokedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRevokeAPIUserAccess handles DELETE /api/apis/:id/users/:user_id
func HandleRevokeAPIUserAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID and user ID from path
	apiID := r.PathValue("id")
	// For tests, check URL path since PathValue may not work in tests
	if apiID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			apiID = parts[3]
		}
	}
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	userID := r.PathValue("user_id")
	// For tests, check URL path since PathValue may not work in tests
	if userID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 6 {
			userID = parts[5]
		}
	}
	if userID == "" {
		sendErrorResponse(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if user is authorized (host user)
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get existing access record
	access, err := db.GetAPIUserAccessByUserID(database, apiID, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "User access record not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve user access: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check if already revoked
	if !access.IsActive {
		sendErrorResponse(w, "Access already revoked", http.StatusBadRequest)
		return
	}

	// Revoke access (soft delete)
	now := time.Now()
	access.IsActive = false
	access.RevokedAt = &now

	if err := db.UpdateAPIUserAccess(database, access); err != nil {
		sendErrorResponse(w, "Failed to revoke user access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIUserAccessResponse{
		ID:          access.ID,
		APIID:       access.APIID,
		UserID:      access.ExternalUserID,
		AccessLevel: access.AccessLevel,
		IsActive:    access.IsActive,
		RevokedAt:   access.RevokedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRestoreAPIUserAccess handles POST /api/apis/:id/users/:user_id/restore
func HandleRestoreAPIUserAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID and user ID from path
	apiID := r.PathValue("id")
	// For tests, check URL path since PathValue may not work in tests
	if apiID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			apiID = parts[3]
		}
	}
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	userID := r.PathValue("user_id")
	// For tests, check URL path since PathValue may not work in tests
	if userID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 6 {
			userID = parts[5]
		}
	}
	if userID == "" {
		sendErrorResponse(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if user is authorized (host user)
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get existing access record
	access, err := db.GetAPIUserAccessByUserID(database, apiID, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "User access record not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve user access: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check if already active
	if access.IsActive {
		sendErrorResponse(w, "Access is already active", http.StatusBadRequest)
		return
	}

	// Restore access
	access.IsActive = true
	access.RevokedAt = nil

	if err := db.UpdateAPIUserAccess(database, access); err != nil {
		sendErrorResponse(w, "Failed to restore user access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIUserAccessResponse{
		ID:          access.ID,
		APIID:       access.APIID,
		UserID:      access.ExternalUserID,
		AccessLevel: access.AccessLevel,
		IsActive:    access.IsActive,
		RevokedAt:   access.RevokedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
