package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestDocumentAssociationCRUD tests the Create, Read, Update, and Delete operations
// for document associations
func TestDocumentAssociationCRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)

	// Create test entity IDs
	apiID := uuid.New().String()
	requestID := uuid.New().String()

	// Insert test API and request records
	_, err := db.Exec(`
		INSERT INTO apis (id, name, host_user_id)
		VALUES (?, ?, ?)`,
		apiID, "Test API", "test_host")
	assert.NoError(t, err, "Failed to insert test API")

	_, err = db.Exec(`
		INSERT INTO api_requests (id, api_name, status, requester_id)
		VALUES (?, ?, ?, ?)`,
		requestID, "Test API", "pending", "test_requester")
	assert.NoError(t, err, "Failed to insert test API request")

	// Test document association creation and retrieval
	t.Run("CreateAndGetDocumentAssociation", func(t *testing.T) {
		// Create a document association for API
		apiDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: "test_api_doc.pdf",
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now().Round(time.Millisecond), // Round to avoid microsecond comparison issues
		}

		err := CreateDocumentAssociation(db, apiDocAssoc)
		assert.NoError(t, err, "Failed to create API document association")

		// Create a document association for Request
		reqDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: "test_request_doc.txt",
			EntityID:         requestID,
			EntityType:       "request",
			CreatedAt:        time.Now().Round(time.Millisecond),
		}

		err = CreateDocumentAssociation(db, reqDocAssoc)
		assert.NoError(t, err, "Failed to create Request document association")

		// Retrieve the API document association
		retrievedAPIDoc, err := GetDocumentAssociation(db, apiDocAssoc.ID)
		assert.NoError(t, err, "Failed to retrieve API document association")
		assert.Equal(t, apiDocAssoc.ID, retrievedAPIDoc.ID, "Document ID mismatch")
		assert.Equal(t, apiDocAssoc.DocumentFilename, retrievedAPIDoc.DocumentFilename, "Document filename mismatch")
		assert.Equal(t, apiDocAssoc.EntityID, retrievedAPIDoc.EntityID, "Entity ID mismatch")
		assert.Equal(t, apiDocAssoc.EntityType, retrievedAPIDoc.EntityType, "Entity type mismatch")
		assert.WithinDuration(t, apiDocAssoc.CreatedAt, retrievedAPIDoc.CreatedAt, time.Second, "Creation time mismatch")

		// Retrieve the Request document association
		retrievedReqDoc, err := GetDocumentAssociation(db, reqDocAssoc.ID)
		assert.NoError(t, err, "Failed to retrieve Request document association")
		assert.Equal(t, reqDocAssoc.ID, retrievedReqDoc.ID, "Document ID mismatch")
		assert.Equal(t, reqDocAssoc.DocumentFilename, retrievedReqDoc.DocumentFilename, "Document filename mismatch")
		assert.Equal(t, reqDocAssoc.EntityID, retrievedReqDoc.EntityID, "Entity ID mismatch")
		assert.Equal(t, reqDocAssoc.EntityType, retrievedReqDoc.EntityType, "Entity type mismatch")
		assert.WithinDuration(t, reqDocAssoc.CreatedAt, retrievedReqDoc.CreatedAt, time.Second, "Creation time mismatch")
	})

	// Test document association duplication prevention
	t.Run("CreateDuplicateDocumentAssociation", func(t *testing.T) {
		// Create a document association
		docAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: "test_duplicate_doc.pdf",
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err := CreateDocumentAssociation(db, docAssoc)
		assert.NoError(t, err, "Failed to create initial document association")

		// Try to create a duplicate with the same filename, entity ID, and entity type
		duplicateAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),       // Different ID
			DocumentFilename: docAssoc.DocumentFilename, // Same filename
			EntityID:         docAssoc.EntityID,         // Same entity ID
			EntityType:       docAssoc.EntityType,       // Same entity type
			CreatedAt:        time.Now(),
		}

		err = CreateDocumentAssociation(db, duplicateAssoc)
		assert.Error(t, err, "Expected error when creating duplicate document association")
		assert.Contains(t, err.Error(), "already associated", "Error should indicate duplicate association")
	})

	// Test transaction-based creation
	t.Run("CreateDocumentAssociationTx", func(t *testing.T) {
		// Start a transaction
		tx, err := db.Begin()
		assert.NoError(t, err, "Failed to begin transaction")

		// Create a document association within the transaction
		docAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: "test_tx_doc.pdf",
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err = CreateDocumentAssociationTx(tx, docAssoc)
		assert.NoError(t, err, "Failed to create document association in transaction")

		// Commit the transaction
		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify the document association was created
		retrievedDoc, err := GetDocumentAssociation(db, docAssoc.ID)
		assert.NoError(t, err, "Failed to retrieve document association")
		assert.Equal(t, docAssoc.ID, retrievedDoc.ID, "Document ID mismatch")
	})

	// Test listing document associations by entity
	t.Run("ListDocumentAssociationsByEntity", func(t *testing.T) {
		// Create several document associations for the same API
		for i := 0; i < 3; i++ {
			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: fmt.Sprintf("test_api_doc_%d.pdf", i),
				EntityID:         apiID,
				EntityType:       "api",
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db, docAssoc)
			assert.NoError(t, err, "Failed to create API document association")
		}

		// Get all document associations for the API
		associations, count, err := GetDocumentAssociationsByEntity(db, "api", apiID)
		assert.NoError(t, err, "Failed to retrieve API document associations")
		assert.GreaterOrEqual(t, count, 3, "Expected at least 3 document associations")
		assert.GreaterOrEqual(t, len(associations), 3, "Expected at least 3 document associations")

		// Verify all retrieved associations are for the API
		for _, assoc := range associations {
			assert.Equal(t, apiID, assoc.EntityID, "Entity ID mismatch")
			assert.Equal(t, "api", assoc.EntityType, "Entity type mismatch")
		}
	})

	// Test pagination in listing associations
	t.Run("ListDocumentAssociationsPagination", func(t *testing.T) {
		// Create a known number of associations
		totalAssocs := 5
		for i := 0; i < totalAssocs; i++ {
			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: fmt.Sprintf("test_pagination_doc_%d.pdf", i),
				EntityID:         requestID,
				EntityType:       "request",
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db, docAssoc)
			assert.NoError(t, err, "Failed to create document association")
		}

		// Test pagination with limit 2, offset 0
		associationsPage1, _, err := ListDocumentAssociations(db, 2, 0)
		assert.NoError(t, err, "Failed to retrieve first page of document associations")
		assert.Equal(t, 2, len(associationsPage1), "Expected 2 associations on first page")

		// Test pagination with limit 2, offset 2
		associationsPage2, _, err := ListDocumentAssociations(db, 2, 2)
		assert.NoError(t, err, "Failed to retrieve second page of document associations")
		assert.Equal(t, 2, len(associationsPage2), "Expected 2 associations on second page")

		// Verify the associations on page 1 and page 2 are different
		assert.NotEqual(t, associationsPage1[0].ID, associationsPage2[0].ID, "Expected different associations on different pages")
	})

	// Test getting all associations for a document
	t.Run("GetAllAssociationsForDocument", func(t *testing.T) {
		// Create a document filename that will be associated with multiple entities
		sharedFilename := "shared_document.pdf"

		// Create an association with the API
		apiAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err := CreateDocumentAssociation(db, apiAssoc)
		assert.NoError(t, err, "Failed to create API document association")

		// Create an association with the Request
		reqAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         requestID,
			EntityType:       "request",
			CreatedAt:        time.Now(),
		}

		err = CreateDocumentAssociation(db, reqAssoc)
		assert.NoError(t, err, "Failed to create Request document association")

		// Get all associations for the shared document
		associations, err := GetAllAssociationsForDocument(db, sharedFilename)
		assert.NoError(t, err, "Failed to retrieve all associations for document")
		assert.Equal(t, 2, len(associations), "Expected 2 associations for the shared document")

		// Verify the associations include both the API and Request
		entityIDs := map[string]bool{apiID: false, requestID: false}
		entityTypes := map[string]bool{"api": false, "request": false}

		for _, assoc := range associations {
			assert.Equal(t, sharedFilename, assoc.DocumentFilename, "Document filename mismatch")
			entityIDs[assoc.EntityID] = true
			entityTypes[assoc.EntityType] = true
		}

		assert.True(t, entityIDs[apiID], "Missing API association")
		assert.True(t, entityIDs[requestID], "Missing Request association")
		assert.True(t, entityTypes["api"], "Missing API entity type")
		assert.True(t, entityTypes["request"], "Missing Request entity type")
	})

	// Test deleting a document association
	t.Run("DeleteDocumentAssociation", func(t *testing.T) {
		// Create a document association
		docAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: "document_to_delete.pdf",
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err := CreateDocumentAssociation(db, docAssoc)
		assert.NoError(t, err, "Failed to create document association")

		// Delete the association
		err = DeleteDocumentAssociation(db, docAssoc.ID)
		assert.NoError(t, err, "Failed to delete document association")

		// Verify the association is deleted
		_, err = GetDocumentAssociation(db, docAssoc.ID)
		assert.Error(t, err, "Expected error when retrieving deleted association")
		assert.Equal(t, ErrNotFound, err, "Expected ErrNotFound")
	})

	// Test deleting all associations for a document
	t.Run("DeleteAllDocumentAssociationsByFilename", func(t *testing.T) {
		// Create a document filename that will be associated with multiple entities
		sharedFilename := fmt.Sprintf("document_to_delete_all_%s.pdf", uuid.New().String())

		// Create two associations with the same filename but different entity types
		// Using unique entities to avoid duplicate association errors

		// Create API association
		apiDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err := CreateDocumentAssociation(db, apiDocAssoc)
		assert.NoError(t, err, "Failed to create API document association")

		// Create Request association
		reqDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         requestID,
			EntityType:       "request",
			CreatedAt:        time.Now(),
		}

		err = CreateDocumentAssociation(db, reqDocAssoc)
		assert.NoError(t, err, "Failed to create Request document association")

		// Verify the associations exist
		associations, err := GetAllAssociationsForDocument(db, sharedFilename)
		assert.NoError(t, err, "Failed to retrieve associations before deletion")
		assert.Equal(t, 2, len(associations), "Expected 2 associations before deletion")

		// Delete all associations for the document
		err = DeleteAllDocumentAssociationsByFilename(db, sharedFilename)
		assert.NoError(t, err, "Failed to delete all document associations")

		// Verify all associations are deleted
		associations, err = GetAllAssociationsForDocument(db, sharedFilename)
		assert.NoError(t, err, "Failed to query associations after deletion")
		assert.Empty(t, associations, "Expected no associations after deletion")
	})

	// Test transaction-based deletion
	t.Run("DeleteAllDocumentAssociationsByFilenameTx", func(t *testing.T) {
		// Create a document filename that will be associated with multiple entities
		// Use a unique filename to avoid conflicts with other tests
		sharedFilename := fmt.Sprintf("document_to_delete_tx_%s.pdf", uuid.New().String())

		// Create API association
		apiDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now(),
		}

		err := CreateDocumentAssociation(db, apiDocAssoc)
		assert.NoError(t, err, "Failed to create API document association")

		// Create Request association
		reqDocAssoc := &DocumentAssociation{
			ID:               uuid.New().String(),
			DocumentFilename: sharedFilename,
			EntityID:         requestID,
			EntityType:       "request",
			CreatedAt:        time.Now(),
		}

		err = CreateDocumentAssociation(db, reqDocAssoc)
		assert.NoError(t, err, "Failed to create Request document association")

		// Get a count before we start to verify we have documents
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM document_associations WHERE document_filename = ?", sharedFilename).Scan(&count)
		assert.NoError(t, err, "Failed to count documents before transaction")
		assert.Equal(t, 2, count, "Expected 2 documents before transaction")

		// Start a transaction
		tx, err := db.Begin()
		assert.NoError(t, err, "Failed to begin transaction")

		// Delete all associations in the transaction
		err = DeleteAllDocumentAssociationsByFilenameTx(tx, sharedFilename)
		assert.NoError(t, err, "Failed to delete document associations in transaction")

		// Commit immediately to avoid transaction locking issues
		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify the documents are gone
		err = db.QueryRow("SELECT COUNT(*) FROM document_associations WHERE document_filename = ?", sharedFilename).Scan(&count)
		assert.NoError(t, err, "Failed to count documents after transaction")
		assert.Equal(t, 0, count, "Expected 0 documents after transaction")
	})

	// Test copying document associations between requests
	t.Run("CopyDocumentsFromRequest", func(t *testing.T) {
		// Create a source request ID
		sourceRequestID := uuid.New().String()

		// Insert the source request
		_, err := db.Exec(`
			INSERT INTO api_requests (id, api_name, status, requester_id)
			VALUES (?, ?, ?, ?)`,
			sourceRequestID, "Source Request", "pending", "test_requester")
		assert.NoError(t, err, "Failed to insert source API request")

		// Create document associations for the source request
		for i := 0; i < 3; i++ {
			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: fmt.Sprintf("source_doc_%d.pdf", i),
				EntityID:         sourceRequestID,
				EntityType:       "request",
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db, docAssoc)
			assert.NoError(t, err, "Failed to create document association for source request")
		}

		// Create a target request ID
		targetRequestID := uuid.New().String()

		// Insert the target request
		_, err = db.Exec(`
			INSERT INTO api_requests (id, api_name, status, requester_id)
			VALUES (?, ?, ?, ?)`,
			targetRequestID, "Target Request", "pending", "test_requester")
		assert.NoError(t, err, "Failed to insert target API request")

		// Start a transaction
		tx, err := db.Begin()
		assert.NoError(t, err, "Failed to begin transaction")

		// Copy documents from source to target request
		err = CopyDocumentsFromRequest(tx, sourceRequestID, targetRequestID)
		assert.NoError(t, err, "Failed to copy documents from request")

		// Commit the transaction
		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify the target request has the same number of document associations
		targetAssocs, _, err := GetDocumentAssociationsByEntity(db, "request", targetRequestID)
		assert.NoError(t, err, "Failed to retrieve target request document associations")
		assert.Equal(t, 3, len(targetAssocs), "Target request should have 3 document associations")

		// Verify the document filenames match
		sourceAssocs, _, err := GetDocumentAssociationsByEntity(db, "request", sourceRequestID)
		assert.NoError(t, err, "Failed to retrieve source request document associations")

		sourceFilenames := make(map[string]bool)
		for _, assoc := range sourceAssocs {
			sourceFilenames[assoc.DocumentFilename] = true
		}

		for _, assoc := range targetAssocs {
			assert.True(t, sourceFilenames[assoc.DocumentFilename], "Target document filename should match a source filename")
			assert.Equal(t, targetRequestID, assoc.EntityID, "Target document should be associated with target request")
			assert.Equal(t, "request", assoc.EntityType, "Target document should have entity type 'request'")
		}
	})

	// Test copying document associations from request to API
	t.Run("CopyDocumentsFromRequestToAPI", func(t *testing.T) {
		// Create a source request ID
		sourceRequestID := uuid.New().String()

		// Insert the source request
		_, err := db.Exec(`
			INSERT INTO api_requests (id, api_name, status, requester_id)
			VALUES (?, ?, ?, ?)`,
			sourceRequestID, "Source Request", "pending", "test_requester")
		assert.NoError(t, err, "Failed to insert source API request")

		// Create document associations for the source request
		for i := 0; i < 2; i++ {
			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: fmt.Sprintf("request_to_api_doc_%d.pdf", i),
				EntityID:         sourceRequestID,
				EntityType:       "request",
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db, docAssoc)
			assert.NoError(t, err, "Failed to create document association for source request")
		}

		// Create a target API ID
		targetAPIID := uuid.New().String()

		// Insert the target API
		_, err = db.Exec(`
			INSERT INTO apis (id, name, host_user_id)
			VALUES (?, ?, ?)`,
			targetAPIID, "Target API", "test_host")
		assert.NoError(t, err, "Failed to insert target API")

		// Start a transaction
		tx, err := db.Begin()
		assert.NoError(t, err, "Failed to begin transaction")

		// Copy documents from request to API
		err = CopyDocumentsFromRequestToAPI(tx, sourceRequestID, targetAPIID)
		assert.NoError(t, err, "Failed to copy documents from request to API")

		// Commit the transaction
		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify the target API has the same number of document associations
		targetAssocs, _, err := GetDocumentAssociationsByEntity(db, "api", targetAPIID)
		assert.NoError(t, err, "Failed to retrieve target API document associations")
		assert.Equal(t, 2, len(targetAssocs), "Target API should have 2 document associations")

		// Verify the document filenames match but entity type is different
		sourceAssocs, _, err := GetDocumentAssociationsByEntity(db, "request", sourceRequestID)
		assert.NoError(t, err, "Failed to retrieve source request document associations")

		sourceFilenames := make(map[string]bool)
		for _, assoc := range sourceAssocs {
			sourceFilenames[assoc.DocumentFilename] = true
			assert.Equal(t, "request", assoc.EntityType, "Source document should have entity type 'request'")
		}

		for _, assoc := range targetAssocs {
			assert.True(t, sourceFilenames[assoc.DocumentFilename], "Target document filename should match a source filename")
			assert.Equal(t, targetAPIID, assoc.EntityID, "Target document should be associated with target API")
			assert.Equal(t, "api", assoc.EntityType, "Target document should have entity type 'api'")
		}
	})
}
