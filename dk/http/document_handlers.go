package http

import (
	"context"
	"dk/core"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// HandleGetDocuments handles GET /api/documents
func HandleGetDocuments(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	entityType := r.URL.Query().Get("entity_type")
	entityID := r.URL.Query().Get("entity_id")

	includeDeleted, _ := strconv.ParseBool(r.URL.Query().Get("is_deleted"))

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

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get document associations based on filters
	var associations []*db.DocumentAssociation
	var total int

	if entityType != "" && entityID != "" {
		// Get associations for a specific entity
		associations, total, err = db.GetDocumentAssociationsByEntity(database, entityType, entityID)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve document associations: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Get all associations with pagination
		associations, total, err = db.ListDocumentAssociations(database, limit, offset)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve document associations: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Convert associations to document references
	documents := make([]DocumentRef, 0, len(associations))

	// We don't need to access the chromem collection directly here
	// The core.GetDocument function will handle that for us

	for _, assoc := range associations {
		// Get document metadata from RAG system
		doc, err := core.GetDocument(ctx, "file", assoc.DocumentFilename, 1)
		if err != nil {
			// Log error but continue
			utils.LogError(ctx, "Failed to retrieve document %s: %v", assoc.DocumentFilename, err)
			continue
		}

		if doc == nil {
			// Document might have been deleted from RAG system
			continue
		}

		// Check if document is marked as deleted and filter accordingly
		if deletedVal, ok := doc.Metadata["is_deleted"]; ok && deletedVal == "true" {
			if !includeDeleted {
				// Skip deleted documents unless explicitly requested
				continue
			}
		}

		// We're not using deletion_date in the response currently, but we could add it
		// to the DocumentRef struct in the future if needed

		// Get document size (approximation based on content length)
		sizeBytes := len(doc.Content)

		// Parse uploaded date
		uploadedAt := time.Now() // Default to now if not found
		if dateStr, ok := doc.Metadata["date"]; ok && dateStr != "" {
			if parsedDate, err := time.Parse("Jan 2, 2006, 03:04 PM", dateStr); err == nil {
				uploadedAt = parsedDate
			}
		}

		docRef := DocumentRef{
			ID:         assoc.ID,
			Name:       doc.FileName,
			Type:       DocumentType(doc.FileName),
			UploadedAt: uploadedAt,
			SizeBytes:  sizeBytes,
		}

		documents = append(documents, docRef)
	}

	response := DocumentListResponse{
		Total:     total,
		Limit:     limit,
		Offset:    offset,
		Documents: documents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetDocument handles GET /api/documents/:id
func HandleGetDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get document ID from path
	docID := r.PathValue("id")
	if docID == "" {
		sendErrorResponse(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the document association
	assoc, err := db.GetDocumentAssociation(database, docID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve document association: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get document content from RAG system
	doc, err := core.GetDocument(ctx, "file", assoc.DocumentFilename, 1)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve document: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if doc == nil {
		sendErrorResponse(w, "Document not found in vector database", http.StatusNotFound)
		return
	}

	// Get all associations for this document
	associations, err := db.GetAllAssociationsForDocument(database, assoc.DocumentFilename)
	if err != nil {
		// Log error but continue
		utils.LogError(ctx, "Failed to retrieve all associations for document %s: %v", assoc.DocumentFilename, err)
	}

	// Convert associations to entity associations
	entityAssociations := make([]EntityAssociation, 0, len(associations))
	for _, a := range associations {
		entityAssoc := EntityAssociation{
			EntityID:   a.EntityID,
			EntityType: a.EntityType,
			CreatedAt:  a.CreatedAt,
		}
		entityAssociations = append(entityAssociations, entityAssoc)
	}

	// Check if document is marked as deleted
	var isDeleted bool
	if deletedVal, ok := doc.Metadata["is_deleted"]; ok && deletedVal == "true" {
		isDeleted = true
	}

	// Get document size (approximation based on content length)
	sizeBytes := len(doc.Content)

	// Parse uploaded date
	uploadedAt := time.Now() // Default to now if not found
	if dateStr, ok := doc.Metadata["date"]; ok && dateStr != "" {
		if parsedDate, err := time.Parse("Jan 2, 2006, 03:04 PM", dateStr); err == nil {
			uploadedAt = parsedDate
		}
	}

	// Get uploader ID if available
	uploaderID := ""
	if id, ok := doc.Metadata["uploader_id"]; ok {
		uploaderID = id
	}

	// Get content type if available
	contentType := InferContentType(doc.FileName)

	// Create response
	response := DocumentDetailResponse{
		ID:           assoc.ID,
		Name:         doc.FileName,
		Type:         DocumentType(doc.FileName),
		ContentType:  contentType,
		SizeBytes:    sizeBytes,
		Content:      doc.Content,
		UploadedAt:   uploadedAt,
		UploaderID:   uploaderID,
		IsDeleted:    isDeleted,
		DeletionDate: nil, // Not deleted
		Associations: entityAssociations,
		Metadata:     doc.Metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUploadDocument handles POST /api/documents
func HandleUploadDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Maximum upload size is 10 MB
	const maxUploadSize = 10 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// Parse the multipart form with max memory of 32MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		sendErrorResponse(w, "File too large or invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file from the request
	file, header, err := r.FormFile("file")
	if err != nil {
		sendErrorResponse(w, "Failed to get file from request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read the file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		sendErrorResponse(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the filename
	filename := header.Filename

	// Get optional entity association parameters
	entityType := r.FormValue("entity_type")
	entityID := r.FormValue("entity_id")

	// Get user ID from context
	userID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		userID = "local-user"
	}

	// Create metadata for the document
	metadata := map[string]string{
		"uploader_id":  userID,
		"is_deleted":   "false",
		"content_type": InferContentType(filename),
	}

	// Add the document to the RAG system
	if err := core.AddDocument(ctx, filename, string(fileContent), true, metadata); err != nil {
		sendErrorResponse(w, "Failed to add document to vector database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create database connection
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Create document association if entity type and ID are provided
	var association *db.DocumentAssociation
	if entityType != "" && entityID != "" {
		// Validate entity type
		if entityType != "api" && entityType != "request" {
			sendErrorResponse(w, "Invalid entity type. Must be 'api' or 'request'", http.StatusBadRequest)
			return
		}

		// Create the association
		association = &db.DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: filename,
			EntityID:         entityID,
			EntityType:       entityType,
			CreatedAt:        time.Now(),
		}

		if err := db.CreateDocumentAssociation(database, association); err != nil {
			// Log error but don't fail the upload - the document is already in the RAG system
			utils.LogError(ctx, "Failed to create document association: %v", err)
		}
	} else {
		// Create a placeholder association for the document
		association = &db.DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: filename,
			EntityID:         "unassociated",
			EntityType:       "unassociated",
			CreatedAt:        time.Now(),
		}
	}

	// Prepare response with document details
	doc, err := core.GetDocument(ctx, "file", filename, 1)
	if err != nil || doc == nil {
		// Should not happen since we just added the document, but handle it gracefully
		sendErrorResponse(w, "Document was uploaded but could not be retrieved for details", http.StatusInternalServerError)
		return
	}

	// Return success with document details
	response := DocumentRef{
		ID:         association.ID,
		Name:       filename,
		Type:       DocumentType(filename),
		UploadedAt: time.Now(),
		SizeBytes:  len(fileContent),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleAssociateDocument handles POST /api/documents/associate
func HandleAssociateDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req DocumentAssociateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.DocumentID == "" {
		sendErrorResponse(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	if req.EntityID == "" {
		sendErrorResponse(w, "Entity ID is required", http.StatusBadRequest)
		return
	}

	if req.EntityType != "api" && req.EntityType != "request" {
		sendErrorResponse(w, "Entity type must be 'api' or 'request'", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the document association to get the filename
	existingAssoc, err := db.GetDocumentAssociation(database, req.DocumentID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve document: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Create new association with the same filename
	association := &db.DocumentAssociation{
		ID:               uuid.New().String(),
		DocumentFilename: existingAssoc.DocumentFilename,
		EntityID:         req.EntityID,
		EntityType:       req.EntityType,
		CreatedAt:        time.Now(),
	}

	// Create the association
	if err := db.CreateDocumentAssociation(database, association); err != nil {
		sendErrorResponse(w, "Failed to create document association: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(association)
}

// HandleSoftDeleteDocument handles DELETE /api/documents/:id
func HandleSoftDeleteDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get document ID from path
	docID := r.PathValue("id")
	if docID == "" {
		sendErrorResponse(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the document association
	assoc, err := db.GetDocumentAssociation(database, docID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve document association: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get current user ID
	userID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		userID = "local-user"
	}

	// Get document from RAG system
	doc, err := core.GetDocument(ctx, "file", assoc.DocumentFilename, 1)
	if err != nil || doc == nil {
		sendErrorResponse(w, "Document not found in vector database", http.StatusNotFound)
		return
	}

	// Check if user is authorized (document uploader or local user)
	uploaderID, ok := doc.Metadata["uploader_id"]
	if !ok || (uploaderID != userID && userID != "local-user") {
		sendErrorResponse(w, "Unauthorized to delete this document", http.StatusForbidden)
		return
	}

	// Check if document is already deleted
	if isDeleted, ok := doc.Metadata["is_deleted"]; ok && isDeleted == "true" {
		sendErrorResponse(w, "Document is already deleted", http.StatusBadRequest)
		return
	}

	// Update metadata to mark as deleted
	now := time.Now()
	deletionDate := now.Format(time.RFC3339)

	// Create updated metadata
	updatedMetadata := make(map[string]string)
	for k, v := range doc.Metadata {
		updatedMetadata[k] = v
	}
	updatedMetadata["is_deleted"] = "true"
	updatedMetadata["deletion_date"] = deletionDate

	// Update the document in RAG system
	if err := core.UpdateDocument(ctx, assoc.DocumentFilename, doc.Content, updatedMetadata); err != nil {
		sendErrorResponse(w, "Failed to update document: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success with updated document
	response := struct {
		ID           string    `json:"id"`
		IsDeleted    bool      `json:"is_deleted"`
		DeletionDate time.Time `json:"deletion_date"`
	}{
		ID:           docID,
		IsDeleted:    true,
		DeletionDate: now,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRestoreDocument handles POST /api/documents/:id/restore
func HandleRestoreDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get document ID from path
	docID := r.PathValue("id")
	if docID == "" {
		sendErrorResponse(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the document association
	assoc, err := db.GetDocumentAssociation(database, docID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve document association: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get current user ID
	userID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		userID = "local-user"
	}

	// Get document from RAG system
	doc, err := core.GetDocument(ctx, "file", assoc.DocumentFilename, 1)
	if err != nil || doc == nil {
		sendErrorResponse(w, "Document not found in vector database", http.StatusNotFound)
		return
	}

	// Check if user is authorized (document uploader or local user)
	uploaderID, ok := doc.Metadata["uploader_id"]
	if !ok || (uploaderID != userID && userID != "local-user") {
		sendErrorResponse(w, "Unauthorized to restore this document", http.StatusForbidden)
		return
	}

	// Check if document is actually deleted
	if isDeleted, ok := doc.Metadata["is_deleted"]; !ok || isDeleted != "true" {
		sendErrorResponse(w, "Document is not deleted", http.StatusBadRequest)
		return
	}

	// Create updated metadata
	updatedMetadata := make(map[string]string)
	for k, v := range doc.Metadata {
		if k != "is_deleted" && k != "deletion_date" {
			updatedMetadata[k] = v
		}
	}
	updatedMetadata["is_deleted"] = "false"

	// Update the document in RAG system
	if err := core.UpdateDocument(ctx, assoc.DocumentFilename, doc.Content, updatedMetadata); err != nil {
		sendErrorResponse(w, "Failed to update document: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success with updated document
	response := struct {
		ID        string `json:"id"`
		IsDeleted bool   `json:"is_deleted"`
	}{
		ID:        docID,
		IsDeleted: false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandlePermanentDeleteDocument handles DELETE /api/documents/:id/permanent
func HandlePermanentDeleteDocument(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get document ID from path
	docID := r.PathValue("id")
	if docID == "" {
		sendErrorResponse(w, "Document ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the document association
	assoc, err := db.GetDocumentAssociation(database, docID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Document not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve document association: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get current user ID
	userID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		userID = "local-user"
	}

	// Only local user can permanently delete documents
	if userID != "local-user" {
		sendErrorResponse(w, "Only the local user can permanently delete documents", http.StatusForbidden)
		return
	}

	// Start a transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Roll back the transaction if it's not committed

	// Delete all associations for this document
	if err := db.DeleteAllDocumentAssociationsByFilenameTx(tx, assoc.DocumentFilename); err != nil {
		sendErrorResponse(w, "Failed to delete document associations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove the document from the RAG system
	if err := core.RemoveDocument(ctx, assoc.DocumentFilename); err != nil {
		sendErrorResponse(w, "Failed to remove document from vector database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
// Note: Most functions have been moved to document_utils.go
