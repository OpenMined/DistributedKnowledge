package http

import (
	"context"
	"dk/core"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// RagRequest represents the JSON structure for POST /rag requests
type RagRequest struct {
	Filename    string `json:"filename"`
	FileContent string `json:"filecontent"`

	Metadata map[string]string `json:"metadata"` // arbitrary keys & values, all strings
	// Metadata    []string `json:"metadata,omitempty"`
}

// RagResponse represents the structure for GET /rag responses
type RagResponse struct {
	Documents []core.Document `json:"documents"`
}

// ErrorResponse represents the structure for error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// SingleDocumentResponse is returned by GET /rag/{file_name}
type SingleDocumentResponse struct {
	Document core.Document `json:"document"`
}

// PatchRagRequest is used by PATCH /rag
type PatchRagRequest struct {
	Filename    string `json:"filename"`
	FileContent string `json:"filecontent"`
	// Metadata    []string `json:"metadata,omitempty"`

	Metadata map[string]string `json:"metadata"` // arbitrary keys & values, all strings
}

// CountResponse is used by GET /rag/count
type CountResponse struct {
	Count int `json:"count"`
}

// FilterRequest is used by GET /rag/filter
type FilterRequest struct {
	Metadata map[string]string `json:"metadata"`
}

// RagQueryRequest is used by GET /rag with metadata filtering
type RagQueryRequest struct {
	Query      string            `json:"query"`
	NumResults int               `json:"num_results"`
	Metadata   map[string]string `json:"metadata"`
}

// Using utils.TrackerDocuments directly for consistency

// Tracker represents a user's tracker configuration
type Tracker struct {
	ID                 int                    `json:"id,omitempty"`
	UserID             string                 `json:"user_id"`
	TrackerName        string                 `json:"tracker_name"`
	TrackerDescription string                 `json:"tracker_description,omitempty"`
	TrackerVersion     string                 `json:"tracker_version,omitempty"`
	TrackerDocuments   utils.TrackerDocuments `json:"tracker_documents,omitempty"`
	CreatedAt          time.Time              `json:"created_at,omitempty"`
	UpdatedAt          time.Time              `json:"updated_at,omitempty"`
}

// TrackerData represents the data stored for a single tracker
type TrackerData struct {
	TrackerDescription string                 `json:"tracker_description,omitempty"`
	TrackerVersion     string                 `json:"tracker_version,omitempty"`
	TrackerDocuments   utils.TrackerDocuments `json:"tracker_documents,omitempty"`
}

// TrackerListPayload represents the new structure for the POST /tracker endpoint
// where trackers is a map with tracker names as keys and tracker data as values
type TrackerListPayload struct {
	UserID   string                 `json:"user_id"`
	Trackers map[string]TrackerData `json:"trackers"`
}

// API represents a user's API configuration
type API struct {
	ID        int       `json:"id,omitempty"`
	UserID    string    `json:"user_id"`
	APIName   string    `json:"api_name"`
	Documents []string  `json:"documents,omitempty"`
	Policy    string    `json:"policy,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// setupHTTPServer initializes and starts the HTTP server
func SetupHTTPServer(ctx context.Context, port string, dbConn *db.DatabaseConnection) {
	// Create a router with the gorilla/mux package for more flexibility
	router := mux.NewRouter()

	// Add the policy enforcement middleware
	router.Use(PolicyEnforcementMiddleware(dbConn))

	// Register usage tracking handlers
	RegisterUsageTrackingHandlers(router, dbConn)

	// API Management Endpoints

	// API Entities
	router.HandleFunc("/api/apis", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIs(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPI(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/apis", func(w http.ResponseWriter, r *http.Request) {
		HandleCreateAPI(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleUpdateAPI(ctx, w, r)
	}).Methods("PATCH")

	router.HandleFunc("/api/apis/{id}/deprecate", func(w http.ResponseWriter, r *http.Request) {
		HandleDeprecateAPI(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/apis/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleDeleteAPI(ctx, w, r)
	}).Methods("DELETE")

	// Policy Management Endpoints
	router.HandleFunc("/api/policies", func(w http.ResponseWriter, r *http.Request) {
		HandleListPolicies(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetPolicy(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/policies/{id}/apis", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIsByPolicy(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/policies", func(w http.ResponseWriter, r *http.Request) {
		HandleCreatePolicy(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleUpdatePolicy(ctx, w, r)
	}).Methods("PATCH")

	router.HandleFunc("/api/policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleDeletePolicy(ctx, w, r)
	}).Methods("DELETE")

	router.HandleFunc("/api/apis/{id}/policy", func(w http.ResponseWriter, r *http.Request) {
		HandleChangeAPIPolicy(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/apis/{id}/policy/history", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIPolicyHistory(ctx, w, r)
	}).Methods("GET")

	// User Access Management Endpoints
	router.HandleFunc("/api/apis/{id}/users", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIUsers(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/apis/{id}/users", func(w http.ResponseWriter, r *http.Request) {
		HandleGrantAPIAccess(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/apis/{id}/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		HandleUpdateAPIUserAccess(ctx, w, r)
	}).Methods("PATCH")

	router.HandleFunc("/api/apis/{id}/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		HandleRevokeAPIUserAccess(ctx, w, r)
	}).Methods("DELETE")

	router.HandleFunc("/api/apis/{id}/users/{user_id}/restore", func(w http.ResponseWriter, r *http.Request) {
		HandleRestoreAPIUserAccess(ctx, w, r)
	}).Methods("POST")

	// API Request Endpoints
	router.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIRequests(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAPIRequest(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
		HandleCreateAPIRequest(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/requests/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		HandleUpdateAPIRequestStatus(ctx, w, r)
	}).Methods("PATCH")

	router.HandleFunc("/api/requests/{id}/resubmit", func(w http.ResponseWriter, r *http.Request) {
		HandleResubmitAPIRequest(ctx, w, r)
	}).Methods("POST")

	// Document Management Endpoints
	router.HandleFunc("/api/documents", func(w http.ResponseWriter, r *http.Request) {
		HandleGetDocuments(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleGetDocument(ctx, w, r)
	}).Methods("GET")

	router.HandleFunc("/api/documents", func(w http.ResponseWriter, r *http.Request) {
		HandleUploadDocument(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/documents/associate", func(w http.ResponseWriter, r *http.Request) {
		HandleAssociateDocument(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/documents/{id}", func(w http.ResponseWriter, r *http.Request) {
		HandleSoftDeleteDocument(ctx, w, r)
	}).Methods("DELETE")

	router.HandleFunc("/api/documents/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		HandleRestoreDocument(ctx, w, r)
	}).Methods("POST")

	router.HandleFunc("/api/documents/{id}/permanent", func(w http.ResponseWriter, r *http.Request) {
		HandlePermanentDeleteDocument(ctx, w, r)
	}).Methods("DELETE")

	// GET /rag/count - Get the total number of documents in the vector database
	router.HandleFunc("/rag/count", func(w http.ResponseWriter, r *http.Request) {
		chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
		if err != nil {
			sendErrorResponse(w, "Failed to access vector database: "+err.Error(), http.StatusInternalServerError)
			return
		}

		count := chromemCollection.Count()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CountResponse{Count: count})
	}).Methods("GET")

	// GET /rag/{file_name} – fetch one document by exact file name
	router.HandleFunc("/rag/{filename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileName := vars["filename"]
		if fileName == "" {
			sendErrorResponse(w, "File name is required", http.StatusBadRequest)
			return
		}

		doc, err := core.GetDocument(ctx, "file", fileName, 1)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve document: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if doc == nil {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SingleDocumentResponse{Document: *doc})
	}).Methods("GET")

	router.HandleFunc("/rag/{filterField}/{filterValue}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filterField := vars["filterField"]
		filterValue := vars["filterValue"]

		if filterField == "" || filterValue == "" {
			sendErrorResponse(w, "Filter field and value are required", http.StatusBadRequest)
			return
		}

		log.Printf("Looking for documents with %s: %s", filterField, filterValue)

		// Get collection to check document count
		chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
		if err != nil {
			sendErrorResponse(w, "Failed to access vector database: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Use collection count instead of a fixed value to avoid "nResults must be <= number of documents" error
		count := chromemCollection.Count()
		docs, err := core.GetDocuments(ctx, filterField, filterValue, count)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve documents: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(docs) == 0 {
			sendErrorResponse(w, "No documents found with this filter criteria", http.StatusNotFound)
			return
		}
		log.Printf("Found %d documents with %s: %s", len(docs), filterField, filterValue)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RagResponse{Documents: docs})
	}).Methods("GET")

	// PATCH /rag – replace a document's content
	router.HandleFunc("/rag", func(w http.ResponseWriter, r *http.Request) {
		var req PatchRagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Filename == "" || req.FileContent == "" {
			sendErrorResponse(w, "Filename and file content are required", http.StatusBadRequest)
			return
		}

		// Update (remove ‑ then add) the document.
		if err := core.UpdateDocument(ctx, req.Filename, req.FileContent, req.Metadata); err != nil {
			sendErrorResponse(w, "Failed to update document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document updated successfully"})
	}).Methods("PATCH")

	// POST /rag - Add document to vector database
	router.HandleFunc("/rag", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Some user made a request ")
		var req RagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Invalid json body...")
			sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Filename == "" || req.FileContent == "" {
			sendErrorResponse(w, "Filename and file content are required", http.StatusBadRequest)
			return
		}

		if err := core.AddDocument(ctx, req.Filename, req.FileContent, true, req.Metadata); err != nil {
			sendErrorResponse(w, "Failed to add document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document added successfully"})
	}).Methods("POST")

	// GET /rag - Retrieve documents based on query with optional metadata filtering
	router.HandleFunc("/rag", func(w http.ResponseWriter, r *http.Request) {
		// Check content type to determine if it's a JSON request
		contentType := r.Header.Get("Content-Type")
		log.Printf("[HTTP] /rag request received with content-type: %s", contentType)

		if contentType == "application/json" {
			// Handle JSON request with metadata filtering
			var req RagQueryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[HTTP] Error decoding JSON request body: %v", err)
				sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}

			if req.Query == "" {
				sendErrorResponse(w, "Query parameter is required", http.StatusBadRequest)
				return
			}

			// Set default number of results if not specified
			if req.NumResults <= 0 {
				req.NumResults = 5
			}

			// Initialize empty metadata map if not provided
			if req.Metadata == nil {
				req.Metadata = make(map[string]string)
			}

			log.Printf("[HTTP] Processing RAG query: '%s' with numResults: %d and metadata: %v",
				req.Query, req.NumResults, req.Metadata)

			// Retrieve documents with metadata filter
			docs, err := core.RetrieveDocuments(ctx, req.Query, req.NumResults, req.Metadata)
			if err != nil {
				log.Printf("[HTTP] Error retrieving documents: %v", err)

				// Check for specific error conditions
				if strings.Contains(err.Error(), "nResults must be <= number of documents") {
					// Return empty results instead of error
					log.Printf("[HTTP] Returning empty result set for query: %s", req.Query)
					response := RagResponse{Documents: []core.Document{}}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
					return
				}

				sendErrorResponse(w, "Failed to retrieve documents: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[HTTP] Successfully retrieved %d documents for query: '%s'", len(docs), req.Query)
			response := RagResponse{Documents: docs}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// Handle traditional query parameters approach
			query := r.URL.Query().Get("query")
			if query == "" {
				sendErrorResponse(w, "Query parameter is required", http.StatusBadRequest)
				return
			}

			numResultsStr := r.URL.Query().Get("num_results")
			numResults := 5 // default value
			if numResultsStr != "" {
				var err error
				numResults, err = strconv.Atoi(numResultsStr)
				if err != nil || numResults <= 0 {
					sendErrorResponse(w, "Invalid num_results parameter", http.StatusBadRequest)
					return
				}
			}

			// Create an empty metadata map for the URL parameter version
			metadata := make(map[string]string)

			log.Printf("[HTTP] Processing URL-based RAG query: '%s' with numResults: %d", query, numResults)

			docs, err := core.RetrieveDocuments(ctx, query, numResults, metadata)
			if err != nil {
				log.Printf("[HTTP] Error retrieving documents with URL parameters: %v", err)

				// Check for specific error conditions
				if strings.Contains(err.Error(), "nResults must be <= number of documents") {
					// Return empty results instead of error
					log.Printf("[HTTP] Returning empty result set for URL query: %s", query)
					response := RagResponse{Documents: []core.Document{}}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
					return
				}

				sendErrorResponse(w, "Failed to retrieve documents: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[HTTP] Successfully retrieved %d documents for URL query: '%s'", len(docs), query)
			response := RagResponse{Documents: docs}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}).Methods("GET")

	// DELETE /rag - Remove document from vector database
	router.HandleFunc("/rag", func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			sendErrorResponse(w, "Filename parameter is required", http.StatusBadRequest)
			return
		}

		if err := core.RemoveDocument(ctx, filename); err != nil {
			sendErrorResponse(w, "Failed to remove document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document removed successfully"})
	}).Methods("DELETE")

	// POST /rag/toggle-active-metadata - Toggle 'active' metadata field on documents
	router.HandleFunc("/rag/toggle-active-metadata", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			FilterField string `json:"filter_field"`
			FilterValue string `json:"filter_value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if request.FilterField == "" || request.FilterValue == "" {
			sendErrorResponse(w, "filter_field and filter_value are required", http.StatusBadRequest)
			return
		}

		log.Printf("Toggling 'active' metadata on documents with %s: %s", request.FilterField, request.FilterValue)

		if err := core.ToggleActiveMetadata(ctx, request.FilterField, request.FilterValue); err != nil {
			sendErrorResponse(w, "Failed to toggle active metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Active metadata toggled successfully"})
	}).Methods("POST")

	// DELETE /rag/all - Delete all documents from the vector database
	router.HandleFunc("/rag/all", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request to delete all documents from vector database")

		if err := core.DeleteAllDocuments(ctx); err != nil {
			sendErrorResponse(w, "Failed to delete vector database: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "All documents successfully deleted from vector database"})
	}).Methods("DELETE")

	// GET /rag/health - Check health of the vector database
	router.HandleFunc("/rag/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request to check vector database health")

		chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
		if err != nil {
			log.Printf("Failed to access vector database: %v", err)
			sendErrorResponse(w, "Failed to access vector database: "+err.Error(), http.StatusInternalServerError)
			return
		}

		documentCount := chromemCollection.Count()
		err = core.CheckChromemHealth(ctx)

		if err != nil {
			log.Printf("Vector database health check failed: %v", err)
			sendErrorResponse(w, "Vector database health check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "healthy",
			"message":        "Vector database is operational",
			"document_count": documentCount,
		})
	}).Methods("GET")

	// POST /api - Register a new API to the websocket server
	router.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		var api API
		if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
			sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if api.APIName == "" {
			sendErrorResponse(w, "API name is required", http.StatusBadRequest)
			return
		}

		// Convert API to APIInfo to communicate with the websocket server
		apiInfo := utils.APIInfo{
			APIName:   api.APIName,
			Documents: api.Documents,
			Policy:    api.Policy,
		}

		// Register the API with the websocket server
		if err := utils.RegisterAPI(ctx, apiInfo); err != nil {
			sendErrorResponse(w, "Failed to register API: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "API registered successfully"})
	}).Methods("POST")

	// POST /user/trackers - Update the user's tracker list in the websocket server
	router.HandleFunc("/user/trackers", func(w http.ResponseWriter, r *http.Request) {
		var trackerList TrackerListPayload
		if err := json.NewDecoder(r.Body).Decode(&trackerList); err != nil {
			sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(trackerList.Trackers) == 0 {
			sendErrorResponse(w, "At least one tracker must be provided", http.StatusBadRequest)
			return
		}

		// Convert from http.TrackerListPayload to utils.TrackerListPayload
		utilTrackerList := utils.TrackerListPayload{
			UserID:   trackerList.UserID,
			Trackers: make(map[string]utils.TrackerData),
		}

		// Copy each tracker from http to utils struct
		for name, data := range trackerList.Trackers {
			utilTrackerList.Trackers[name] = utils.TrackerData{
				TrackerDescription: data.TrackerDescription,
				TrackerVersion:     data.TrackerVersion,
				TrackerDocuments:   data.TrackerDocuments,
			}
		}

		// Use RegisterTrackerList utility for batch registration
		if err := utils.RegisterTrackerList(ctx, utilTrackerList); err != nil {
			sendErrorResponse(w, "Failed to register tracker list: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "Tracker list updated successfully"})
	}).Methods("POST")

	// POST /remote/message - Send a remote message to peers
	router.HandleFunc("/remote/message", func(w http.ResponseWriter, r *http.Request) {
		HandleSendRemoteMessage(ctx, w, r)
	}).Methods("POST")

	// POST /rag/fix-metadata - Ensure all documents have required metadata fields
	router.HandleFunc("/rag/fix-metadata", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] Received request to fix document metadata")

		stats, err := core.EnsureDocumentMetadata(ctx)
		if err != nil {
			log.Printf("[HTTP] Error fixing document metadata: %v", err)
			sendErrorResponse(w, "Failed to fix document metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[HTTP] Successfully fixed document metadata: %v", stats)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Document metadata fixed successfully",
			"stats":   stats,
		})
	}).Methods("POST")

	// GET or POST /answers - Retrieve answers for a given query string
	router.HandleFunc("/answers", func(w http.ResponseWriter, r *http.Request) {
		HandleGetAnswersByQuery(ctx, w, r)
	}).Methods("GET", "POST")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
}

// sendErrorResponse is a helper function to send error responses
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
