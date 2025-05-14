package http

import (
	"context"
	"database/sql"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// PathParamContextKey is the context key for path parameters
const PathParamContextKey = "pathParams"

// Helper function to get a path parameter from the request context
func getPathParam(r *http.Request, param string) string {
	// Try PathValue first (Go 1.22+)
	if idFromURL := r.PathValue(param); idFromURL != "" {
		return idFromURL
	}

	// Try context value
	if pathParams, ok := r.Context().Value(PathParamContextKey).(map[string]string); ok {
		return pathParams[param]
	}

	// Manual URL parsing as last resort - this mimics the approach in HandleGetAPIUsers
	parts := strings.Split(r.URL.Path, "/")

	// For path patterns like /api/apis/{id}
	if param == "id" && len(parts) >= 4 {
		return parts[3]
	}

	// For path patterns like /api/apis/{id}/users/{user_id}
	if param == "user_id" && len(parts) >= 6 {
		return parts[5]
	}

	return ""
}

// HandleGetAPIs handles GET /api/apis
func HandleGetAPIs(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	externalUserID := r.URL.Query().Get("external_user_id")

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
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the APIs from the database
	apis, total, err := db.ListAPIs(database, status, externalUserID, limit, offset, sort, order)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve APIs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	apiBasicList := make([]APIBasic, 0, len(apis))
	for _, api := range apis {
		// Get external user count
		userCount, err := db.CountAPIExternalUsers(database, api.ID)
		if err != nil {
			sendErrorResponse(w, "Failed to count external users: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Get document count
		docCount, err := db.CountAPIDocuments(database, api.ID)
		if err != nil {
			sendErrorResponse(w, "Failed to count documents: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Get policy if available
		var policyRef *PolicyRef
		if api.PolicyID != nil {
			policy, err := db.GetPolicy(database, *api.PolicyID)
			if err == nil {
				policyRef = &PolicyRef{
					ID:   policy.ID,
					Name: policy.Name,
					Type: policy.Type,
				}
			}
		}

		apiBasic := APIBasic{
			ID:                 api.ID,
			Name:               api.Name,
			Description:        api.Description,
			IsActive:           api.IsActive,
			IsDeprecated:       api.IsDeprecated,
			CreatedAt:          api.CreatedAt,
			UpdatedAt:          api.UpdatedAt,
			Policy:             policyRef,
			ExternalUsersCount: userCount,
			DocumentsCount:     docCount,
		}

		apiBasicList = append(apiBasicList, apiBasic)
	}

	response := APIListResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		APIs:   apiBasicList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetAPI handles GET /api/apis/:id
func HandleGetAPI(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Use our improved getPathParam function to get the API ID
	apiID := getPathParam(r, "id")

	// If not found from path, try context as fallback
	if apiID == "" && r.Context().Value("id") != nil {
		apiID = r.Context().Value("id").(string)
	}

	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the API from the database
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get external users
	users, err := db.GetAPIExternalUsers(database, apiID)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve external users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert users to response format
	userRefs := make([]UserRef, 0, len(users))
	for _, user := range users {
		// In a real implementation, you would fetch user details from your user store
		// For now we'll use placeholder data
		avatar := string(user.ExternalUserID[0])
		if avatar == "" {
			avatar = "U"
		}

		userRef := UserRef{
			ID:          user.ExternalUserID,
			Name:        "User " + user.ExternalUserID, // Placeholder
			Avatar:      avatar,
			AccessLevel: user.AccessLevel,
		}
		userRefs = append(userRefs, userRef)
	}

	// Get associated documents
	documents, err := db.GetAPIDocuments(database, apiID)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve documents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert documents to response format
	documentRefs := make([]DocumentRef, 0, len(documents))
	for _, doc := range documents {
		// For each document filename, get the actual document from RAG system
		// In a real implementation, you would fetch document details from your document store
		docRef := DocumentRef{
			ID:         doc.ID,
			Name:       doc.DocumentFilename,
			Type:       DocumentType(doc.DocumentFilename),
			UploadedAt: doc.CreatedAt,
			SizeBytes:  1024, // Placeholder
		}
		documentRefs = append(documentRefs, docRef)
	}

	// Get policy details if available
	var policyDetail *PolicyDetail
	if api.PolicyID != nil {
		policy, err := db.GetPolicyWithRules(database, *api.PolicyID)
		if err == nil {
			rules := make([]PolicyRuleDetail, 0, len(policy.Rules))
			for _, rule := range policy.Rules {
				ruleDetail := PolicyRuleDetail{
					Type:   rule.RuleType,
					Limit:  rule.LimitValue,
					Period: rule.Period,
					Action: rule.Action,
				}
				rules = append(rules, ruleDetail)
			}

			policyDetail = &PolicyDetail{
				ID:    policy.ID,
				Name:  policy.Name,
				Type:  policy.Type,
				Rules: rules,
			}
		}
	}

	// Get usage statistics
	usageSummary, err := getAPIUsageSummary(database, apiID)
	if err != nil {
		// Log the error but continue, as usage data is not critical
		fmt.Printf("Failed to get API usage summary: %v\n", err)
	}

	response := APIDetailResponse{
		ID:            api.ID,
		Name:          api.Name,
		Description:   api.Description,
		IsActive:      api.IsActive,
		IsDeprecated:  api.IsDeprecated,
		CreatedAt:     api.CreatedAt,
		UpdatedAt:     api.UpdatedAt,
		APIKey:        api.APIKey,
		ExternalUsers: userRefs,
		Documents:     documentRefs,
		Policy:        policyDetail,
		UsageSummary:  usageSummary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCreateAPI handles POST /api/apis
func HandleCreateAPI(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req CreateAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		sendErrorResponse(w, "API name is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get user ID from context or use a default for now
	hostUserID := "local-user" // Default for now
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		hostUserID = userID
	}

	// Start a transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Roll back the transaction if it's not committed

	// Create the API
	api := &db.API{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		HostUserID:  hostUserID,
	}

	// Set policy ID if provided
	if req.PolicyID != "" {
		api.PolicyID = &req.PolicyID
	}

	// Create API record
	if err := db.CreateAPITx(tx, api); err != nil {
		sendErrorResponse(w, "Failed to create API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Associate documents if provided
	for _, docID := range req.DocumentIDs {
		association := &db.DocumentAssociation{
			DocumentFilename: docID,
			EntityID:         api.ID,
			EntityType:       "api",
		}

		if err := db.CreateDocumentAssociationTx(tx, association); err != nil {
			sendErrorResponse(w, "Failed to associate document: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Grant access to external users if provided
	for _, user := range req.ExternalUsers {
		access := &db.APIUserAccess{
			APIID:          api.ID,
			ExternalUserID: user.UserID,
			AccessLevel:    user.AccessLevel,
			GrantedBy:      hostUserID,
			IsActive:       true,
		}

		if err := db.CreateAPIUserAccessTx(tx, access); err != nil {
			sendErrorResponse(w, "Failed to grant user access: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created API
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(api)
}

// HandleUpdateAPI handles PATCH /api/apis/:id
func HandleUpdateAPI(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Use our improved getPathParam function to get the API ID
	apiID := getPathParam(r, "id")

	// If not found from path, try context as fallback
	if apiID == "" && r.Context().Value("id") != nil {
		apiID = r.Context().Value("id").(string)
	}

	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the existing API
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Update the fields that were provided
	if req.Name != nil {
		api.Name = *req.Name
	}

	if req.Description != nil {
		api.Description = *req.Description
	}

	if req.PolicyID != nil {
		api.PolicyID = req.PolicyID
	}

	if req.IsActive != nil {
		api.IsActive = *req.IsActive
	}

	// Update the API in the database
	if err := db.UpdateAPI(database, api); err != nil {
		sendErrorResponse(w, "Failed to update API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If policy was updated, record the change in policy_changes table
	if req.PolicyID != nil {
		// Get user ID from context or use a default for now
		changedBy := "local-user" // Default for now
		if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
			changedBy = userID
		}

		// Get the current time
		now := time.Now()

		policyChange := &db.PolicyChange{
			APIID:         apiID,
			NewPolicyID:   req.PolicyID,
			ChangedBy:     changedBy,
			ChangedAt:     now,
			EffectiveDate: &now,
			ChangeReason:  "API update",
		}

		if err := db.CreatePolicyChange(database, policyChange); err != nil {
			// Log the error but don't fail the request
			fmt.Printf("Failed to record policy change: %v\n", err)
		}
	}

	// Return the updated API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api)
}

// HandleDeprecateAPI handles POST /api/apis/:id/deprecate
func HandleDeprecateAPI(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Use our improved getPathParam function to get the API ID
	apiID := getPathParam(r, "id")

	// If not found from path, try context as fallback
	if apiID == "" && r.Context().Value("id") != nil {
		apiID = r.Context().Value("id").(string)
	}

	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	var req DeprecateAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the existing API
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Update deprecation fields
	api.IsDeprecated = true
	api.DeprecationDate = &req.DeprecationDate
	api.DeprecationMessage = req.DeprecationMessage

	// Update the API in the database
	if err := db.UpdateAPI(database, api); err != nil {
		sendErrorResponse(w, "Failed to deprecate API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the updated API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api)
}

// HandleDeleteAPI handles DELETE /api/apis/:id
func HandleDeleteAPI(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Use our improved getPathParam function to get the API ID
	apiID := getPathParam(r, "id")

	// If not found from path, try other sources as fallbacks
	if apiID == "" {
		// Try to get ID from request context
		if idFromContext := r.Context().Value("id"); idFromContext != nil {
			apiID = idFromContext.(string)
		} else if idFromQuery := r.URL.Query().Get("id"); idFromQuery != "" {
			// Try to get ID from query parameter
			apiID = idFromQuery
		} else if idFromHeader := r.Header.Get("X-API-ID"); idFromHeader != "" {
			// Try to get ID from custom header
			apiID = idFromHeader
		}
	}

	if apiID == "" {
		// Log debug information for troubleshooting
		fmt.Printf("DEBUG: DELETE request missing API ID. URL.Path: %s, Headers: %v, Query: %v\n",
			r.URL.Path, r.Header, r.URL.Query())
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Delete the API
	if err := db.DeleteAPI(database, apiID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to delete API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// Note: Document type function is now provided by DocumentType() in document_utils.go

// getAPIUsageSummary retrieves usage statistics for an API
func getAPIUsageSummary(db *sql.DB, apiID string) (*UsageSummary, error) {
	// For now, return placeholder data since this is just for testing
	summary := &UsageSummary{}

	// Initialize the Today section
	summary.Today.Requests = 10
	summary.Today.Tokens = 5000
	summary.Today.ThrottledRequests = 0
	summary.Today.BlockedRequests = 0

	// Initialize the ThisMonth section
	summary.ThisMonth.Requests = 150
	summary.ThisMonth.Tokens = 75000
	summary.ThisMonth.ThrottledRequests = 5
	summary.ThisMonth.BlockedRequests = 2

	return summary, nil
}

// === API Request Endpoints Handlers ===

// HandleGetAPIRequests handles GET /api/requests
func HandleGetAPIRequests(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	requesterID := r.URL.Query().Get("requester_id")

	// Parse pagination parameters
	limit := 10 // default
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
	sort := "submitted_date" // default
	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		if sortParam == "submitted_date" || sortParam == "api_name" {
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

	// Get the host user ID (local user)
	hostUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		hostUserID = "local-user"
	}

	// Get the requests from the database
	requests, total, err := db.ListAPIRequests(database, status, requesterID, hostUserID, limit, offset, sort, order)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve API requests: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	requestBasicList := make([]APIRequestBasic, 0, len(requests))
	for _, req := range requests {
		// Get document count
		docCount, err := db.CountRequestDocuments(database, req.ID)
		if err != nil {
			// Log error but continue
			utils.LogError(ctx, "Failed to count documents for request %s: %v", req.ID, err)
			docCount = 0
		}

		// Get tracker count
		trackerCount, err := db.CountRequestTrackers(database, req.ID)
		if err != nil {
			// Log error but continue
			utils.LogError(ctx, "Failed to count trackers for request %s: %v", req.ID, err)
			trackerCount = 0
		}

		// In a real implementation, fetch user details from your user store
		// For now, use placeholder data
		requesterName := "User " + req.RequesterID // Placeholder
		avatar := string(req.RequesterID[0])
		if avatar == "" {
			avatar = "U"
		}

		requester := UserRef{
			ID:     req.RequesterID,
			Name:   requesterName,
			Avatar: avatar,
		}

		requestBasic := APIRequestBasic{
			ID:                    req.ID,
			APIName:               req.APIName,
			Description:           req.Description,
			Status:                req.Status,
			SubmissionCount:       req.SubmissionCount,
			SubmittedDate:         req.SubmittedDate,
			Requester:             requester,
			DocumentsCount:        docCount,
			RequiredTrackersCount: trackerCount,
		}

		requestBasicList = append(requestBasicList, requestBasic)
	}

	response := APIRequestListResponse{
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		Requests: requestBasicList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAPIRequest handles GET /api/requests/:id
func HandleGetAPIRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get request ID from path
	requestID := getPathParam(r, "id")
	if requestID == "" {
		sendErrorResponse(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the request from the database
	apiRequest, err := db.GetAPIRequest(database, requestID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API request not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API request: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Check if current user is the host user or the requester
	isAuthorized := currentUserID == "local-user" || currentUserID == apiRequest.RequesterID
	if !isAuthorized {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// In a real implementation, fetch user details from your user store
	// For now, use placeholder data
	requesterName := "User " + apiRequest.RequesterID // Placeholder
	avatar := string(apiRequest.RequesterID[0])
	if avatar == "" {
		avatar = "U"
	}

	requester := UserRef{
		ID:     apiRequest.RequesterID,
		Name:   requesterName,
		Avatar: avatar,
	}

	// Get associated documents
	documents, err := db.GetRequestDocuments(database, requestID)
	if err != nil {
		// Log error but continue
		utils.LogError(ctx, "Failed to get documents for request %s: %v", requestID, err)
	}

	// Convert documents to response format
	documentRefs := make([]DocumentRef, 0, len(documents))
	for _, doc := range documents {
		docRef := DocumentRef{
			ID:         doc.ID,
			Name:       doc.DocumentFilename,
			Type:       DocumentType(doc.DocumentFilename),
			UploadedAt: doc.CreatedAt,
			SizeBytes:  0, // Placeholder
		}
		documentRefs = append(documentRefs, docRef)
	}

	// Get required trackers
	trackers, err := db.GetRequestTrackers(database, requestID)
	if err != nil {
		// Log error but continue
		utils.LogError(ctx, "Failed to get trackers for request %s: %v", requestID, err)
	}

	// Convert trackers to response format
	trackerRefs := make([]TrackerRef, 0, len(trackers))
	for _, tracker := range trackers {
		trackerRef := TrackerRef{
			ID:          tracker.TrackerID,
			Name:        tracker.Name,
			Description: tracker.Description,
		}
		trackerRefs = append(trackerRefs, trackerRef)
	}

	// Get previous request if it exists
	var previousRequest *APIRequestRef
	if apiRequest.PreviousRequestID != nil {
		prevReq, err := db.GetAPIRequest(database, *apiRequest.PreviousRequestID)
		if err == nil {
			previousRequest = &APIRequestRef{
				ID:            prevReq.ID,
				Status:        prevReq.Status,
				SubmittedDate: prevReq.SubmittedDate,
			}
		}
	}

	// Get proposed policy if it exists
	var proposedPolicy *PolicyRef
	if apiRequest.ProposedPolicyID != nil {
		policy, err := db.GetPolicy(database, *apiRequest.ProposedPolicyID)
		if err == nil {
			proposedPolicy = &PolicyRef{
				ID:   policy.ID,
				Name: policy.Name,
				Type: policy.Type,
			}
		}
	}

	response := APIRequestDetailResponse{
		ID:               apiRequest.ID,
		APIName:          apiRequest.APIName,
		Description:      apiRequest.Description,
		SubmittedDate:    apiRequest.SubmittedDate,
		Status:           apiRequest.Status,
		Requester:        requester,
		Documents:        documentRefs,
		RequiredTrackers: trackerRefs,
		DenialReason:     apiRequest.DenialReason,
		DeniedDate:       apiRequest.DeniedDate,
		ApprovedDate:     apiRequest.ApprovedDate,
		SubmissionCount:  apiRequest.SubmissionCount,
		PreviousRequest:  previousRequest,
		ProposedPolicy:   proposedPolicy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCreateAPIRequest handles POST /api/requests
func HandleCreateAPIRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req CreateAPIRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.APIName == "" {
		sendErrorResponse(w, "API name is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the requester ID (external user)
	requesterID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		requesterID = "external-user"
	}

	// Start a transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Roll back the transaction if it's not committed

	// Create the API request
	apiRequest := &db.APIRequest{
		ID:              uuid.New().String(),
		APIName:         req.APIName,
		Description:     req.Description,
		SubmittedDate:   time.Now(),
		Status:          "pending",
		RequesterID:     requesterID,
		SubmissionCount: 1,
	}

	// Set proposed policy ID if provided
	if req.ProposedPolicyID != "" {
		apiRequest.ProposedPolicyID = &req.ProposedPolicyID
	}

	// Create API request record
	if err := db.CreateAPIRequestTx(tx, apiRequest); err != nil {
		sendErrorResponse(w, "Failed to create API request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Associate documents if provided
	for _, docID := range req.DocumentIDs {
		association := &db.DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: docID,
			EntityID:         apiRequest.ID,
			EntityType:       "request",
		}

		if err := db.CreateDocumentAssociationTx(tx, association); err != nil {
			sendErrorResponse(w, "Failed to associate document: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Associate required trackers if provided
	for _, trackerID := range req.RequiredTrackerIDs {
		// Verify tracker exists
		tracker, err := db.GetTracker(tx, trackerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				sendErrorResponse(w, "Tracker not found: "+trackerID, http.StatusBadRequest)
			} else {
				sendErrorResponse(w, "Failed to verify tracker: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Create association
		association := &db.RequestRequiredTracker{
			ID:        uuid.New().String(),
			RequestID: apiRequest.ID,
			TrackerID: tracker.ID,
		}

		if err := db.CreateRequestTrackerTx(tx, association); err != nil {
			sendErrorResponse(w, "Failed to associate tracker: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created request
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiRequest)
}

// HandleUpdateAPIRequestStatus handles PATCH /api/requests/:id/status
func HandleUpdateAPIRequestStatus(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get request ID from path
	requestID := getPathParam(r, "id")
	if requestID == "" {
		sendErrorResponse(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateAPIRequestStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Status != "approved" && req.Status != "denied" {
		sendErrorResponse(w, "Status must be 'approved' or 'denied'", http.StatusBadRequest)
		return
	}

	if req.Status == "approved" && req.PolicyID == "" {
		sendErrorResponse(w, "Policy ID is required for approval", http.StatusBadRequest)
		return
	}

	if req.Status == "denied" && req.DenialReason == "" {
		sendErrorResponse(w, "Denial reason is required for rejection", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the host user ID (local user)
	hostUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		hostUserID = "local-user"
	}

	// Start a transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Roll back the transaction if it's not committed

	// Get the request
	apiRequest, err := db.GetAPIRequestTx(tx, requestID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API request not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API request: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Verify the request is in pending status
	if apiRequest.Status != "pending" {
		sendErrorResponse(w, "Cannot update status of non-pending request", http.StatusBadRequest)
		return
	}

	now := time.Now()

	// Update the request status
	if req.Status == "approved" {
		apiRequest.Status = "approved"
		apiRequest.ApprovedDate = &now

		// If create_api is true, create a new API
		if req.CreateAPI {
			// Create a new API based on the request
			api := &db.API{
				ID:          uuid.New().String(),
				Name:        apiRequest.APIName,
				Description: apiRequest.Description,
				IsActive:    true,
				HostUserID:  hostUserID,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// Set policy ID
			api.PolicyID = &req.PolicyID

			// Create API record
			if err := db.CreateAPITx(tx, api); err != nil {
				sendErrorResponse(w, "Failed to create API: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Copy documents from request to API
			if err := db.CopyDocumentsFromRequestToAPI(tx, requestID, api.ID); err != nil {
				sendErrorResponse(w, "Failed to copy documents: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Grant access to the requester
			access := &db.APIUserAccess{
				ID:             uuid.New().String(),
				APIID:          api.ID,
				ExternalUserID: apiRequest.RequesterID,
				AccessLevel:    "read", // Default to read access
				GrantedBy:      hostUserID,
				GrantedAt:      now,
				IsActive:       true,
			}

			if err := db.CreateAPIUserAccessTx(tx, access); err != nil {
				sendErrorResponse(w, "Failed to grant user access: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Record the policy assignment
			policyChange := &db.PolicyChange{
				ID:            uuid.New().String(),
				APIID:         api.ID,
				NewPolicyID:   &req.PolicyID,
				ChangedBy:     hostUserID,
				ChangedAt:     now,
				EffectiveDate: &now,
				ChangeReason:  "Initial policy assignment during API creation",
			}

			if err := db.CreatePolicyChangeTx(tx, policyChange); err != nil {
				// Log error but continue
				utils.LogError(ctx, "Failed to record policy change: %v", err)
			}
		}
	} else {
		apiRequest.Status = "denied"
		apiRequest.DenialReason = req.DenialReason
		apiRequest.DeniedDate = &now
	}

	// Update the request in the database
	if err := db.UpdateAPIRequestTx(tx, apiRequest); err != nil {
		sendErrorResponse(w, "Failed to update API request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the updated request
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiRequest)
}

// HandleResubmitAPIRequest handles POST /api/requests/:id/resubmit
func HandleResubmitAPIRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get request ID from path
	originalRequestID := getPathParam(r, "id")
	if originalRequestID == "" {
		sendErrorResponse(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	var req ResubmitAPIRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the requester ID (external user)
	requesterID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		requesterID = "external-user"
	}

	// Get the original request
	originalRequest, err := db.GetAPIRequest(database, originalRequestID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Original request not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve original request: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Verify the request is in denied status
	if originalRequest.Status != "denied" {
		sendErrorResponse(w, "Only denied requests can be resubmitted", http.StatusBadRequest)
		return
	}

	// Verify the requester is the original requester
	if originalRequest.RequesterID != requesterID {
		sendErrorResponse(w, "Only the original requester can resubmit a request", http.StatusForbidden)
		return
	}

	// Start a transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Roll back the transaction if it's not committed

	// Create a new request based on the original
	newRequest := &db.APIRequest{
		ID:                uuid.New().String(),
		APIName:           originalRequest.APIName,
		SubmittedDate:     time.Now(),
		Status:            "pending",
		RequesterID:       requesterID,
		SubmissionCount:   originalRequest.SubmissionCount + 1,
		PreviousRequestID: &originalRequestID,
	}

	// Override fields if provided in the request
	if req.Description != "" {
		newRequest.Description = req.Description
	} else {
		newRequest.Description = originalRequest.Description
	}

	// Set proposed policy ID if provided
	if req.ProposedPolicyID != "" {
		newRequest.ProposedPolicyID = &req.ProposedPolicyID
	} else if originalRequest.ProposedPolicyID != nil {
		newRequest.ProposedPolicyID = originalRequest.ProposedPolicyID
	}

	// Create API request record
	if err := db.CreateAPIRequestTx(tx, newRequest); err != nil {
		sendErrorResponse(w, "Failed to create API request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle document associations
	if len(req.DocumentIDs) > 0 {
		// Use provided document IDs
		for _, docID := range req.DocumentIDs {
			association := &db.DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: docID,
				EntityID:         newRequest.ID,
				EntityType:       "request",
			}

			if err := db.CreateDocumentAssociationTx(tx, association); err != nil {
				sendErrorResponse(w, "Failed to associate document: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Copy documents from the original request
		if err := db.CopyDocumentsFromRequest(tx, originalRequestID, newRequest.ID); err != nil {
			sendErrorResponse(w, "Failed to copy documents: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Handle tracker associations
	if len(req.RequiredTrackerIDs) > 0 {
		// Use provided tracker IDs
		for _, trackerID := range req.RequiredTrackerIDs {
			// Verify tracker exists
			tracker, err := db.GetTracker(tx, trackerID)
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					sendErrorResponse(w, "Tracker not found: "+trackerID, http.StatusBadRequest)
				} else {
					sendErrorResponse(w, "Failed to verify tracker: "+err.Error(), http.StatusInternalServerError)
				}
				return
			}

			// Create association
			association := &db.RequestRequiredTracker{
				ID:        uuid.New().String(),
				RequestID: newRequest.ID,
				TrackerID: tracker.ID,
			}

			if err := db.CreateRequestTrackerTx(tx, association); err != nil {
				sendErrorResponse(w, "Failed to associate tracker: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Copy trackers from the original request
		if err := db.CopyTrackersFromRequest(tx, originalRequestID, newRequest.ID); err != nil {
			sendErrorResponse(w, "Failed to copy trackers: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created request
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newRequest)
}
