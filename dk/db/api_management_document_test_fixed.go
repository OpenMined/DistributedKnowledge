package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestDocumentAssociationFixed tests the Create, Read, Update, and Delete operations
// for document associations with improved handling of SQLite concurrency
func TestDocumentAssociationFixed(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database - create a fresh connection for each test
	db, err := OpenTestDB()
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Initialize tables
	err = InitAPIManagementTables(db.DB)
	if err != nil {
		t.Fatalf("Failed to initialize tables: %v", err)
	}

	// Create test entity IDs
	apiID := uuid.New().String()
	requestID := uuid.New().String()

	// Insert test API and request records
	_, err = db.Exec(`
		INSERT INTO apis (id, name, host_user_id)
		VALUES (?, ?, ?)`,
		apiID, "Test API", "test_host")
	assert.NoError(t, err, "Failed to insert test API")

	_, err = db.Exec(`
		INSERT INTO api_requests (id, api_name, status, requester_id)
		VALUES (?, ?, ?, ?)`,
		requestID, "Test API", "pending", "test_requester")
	assert.NoError(t, err, "Failed to insert test API request")

	// Tests for both basic document operations and transaction behavior
	t.Run("BasicDocumentOperations", func(t *testing.T) {
		// Test document association creation
		docID := uuid.New().String()
		filename := fmt.Sprintf("test_doc_%s.pdf", uuid.New().String())

		docAssoc := &DocumentAssociation{
			ID:               docID,
			DocumentFilename: filename,
			EntityID:         apiID,
			EntityType:       "api",
			CreatedAt:        time.Now().Round(time.Millisecond),
		}

		err := CreateDocumentAssociation(db.DB, docAssoc)
		assert.NoError(t, err, "Failed to create document association")

		// Test document retrieval
		retrievedDoc, err := GetDocumentAssociation(db.DB, docID)
		assert.NoError(t, err, "Failed to retrieve document association")
		assert.Equal(t, docID, retrievedDoc.ID, "Document ID mismatch")
		assert.Equal(t, filename, retrievedDoc.DocumentFilename, "Document filename mismatch")

		// Test document deletion
		err = DeleteDocumentAssociation(db.DB, docID)
		assert.NoError(t, err, "Failed to delete document association")

		// Verify deletion
		_, err = GetDocumentAssociation(db.DB, docID)
		assert.Error(t, err, "Expected error when retrieving deleted association")
	})

	t.Run("TransactionHandling", func(t *testing.T) {
		// Create a unique filename for this test
		sharedFilename := fmt.Sprintf("tx_doc_%s.pdf", uuid.New().String())

		// Create documents to test with
		for i := 0; i < 2; i++ {
			entityID := apiID
			entityType := "api"
			if i > 0 {
				entityID = requestID
				entityType = "request"
			}

			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: sharedFilename,
				EntityID:         entityID,
				EntityType:       entityType,
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db.DB, docAssoc)
			assert.NoError(t, err, "Failed to create document association")
		}

		// Verify documents exist
		initialDocs, err := GetAllAssociationsForDocument(db.DB, sharedFilename)
		assert.NoError(t, err, "Failed to get initial documents")
		assert.Equal(t, 2, len(initialDocs), "Expected 2 initial documents")

		// Start a transaction
		tx, err := db.DB.Begin()
		assert.NoError(t, err, "Failed to start transaction")

		// Use transaction to delete documents
		err = DeleteAllDocumentAssociationsByFilenameTx(tx, sharedFilename)
		assert.NoError(t, err, "Failed to delete in transaction")

		// Commit immediately to avoid lock issues
		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify documents are gone
		finalDocs, err := GetAllAssociationsForDocument(db.DB, sharedFilename)
		assert.NoError(t, err, "Failed to get final document list")
		assert.Empty(t, finalDocs, "Expected all documents to be deleted")
	})

	t.Run("CopyDocumentTest", func(t *testing.T) {
		// Create source and target entities
		sourceID := uuid.New().String()
		targetID := uuid.New().String()

		// Create source records
		_, err := db.Exec(`
			INSERT INTO api_requests (id, api_name, status, requester_id)
			VALUES (?, ?, ?, ?)`,
			sourceID, "Source", "pending", "test_requester")
		assert.NoError(t, err, "Failed to insert source request")

		_, err = db.Exec(`
			INSERT INTO api_requests (id, api_name, status, requester_id)
			VALUES (?, ?, ?, ?)`,
			targetID, "Target", "pending", "test_requester")
		assert.NoError(t, err, "Failed to insert target request")

		// Create document associations for source
		for i := 0; i < 2; i++ {
			docAssoc := &DocumentAssociation{
				ID:               uuid.New().String(),
				DocumentFilename: fmt.Sprintf("copy_test_%d.pdf", i),
				EntityID:         sourceID,
				EntityType:       "request",
				CreatedAt:        time.Now(),
			}

			err := CreateDocumentAssociation(db.DB, docAssoc)
			assert.NoError(t, err, "Failed to create source document")
		}

		// Copy documents using a transaction that we commit immediately
		tx, err := db.DB.Begin()
		assert.NoError(t, err, "Failed to begin transaction")

		err = CopyDocumentsFromRequest(tx, sourceID, targetID)
		assert.NoError(t, err, "Failed to copy documents")

		err = tx.Commit()
		assert.NoError(t, err, "Failed to commit transaction")

		// Verify copied documents
		targetDocs, _, err := GetDocumentAssociationsByEntity(db.DB, "request", targetID)
		assert.NoError(t, err, "Failed to get target documents")
		assert.Equal(t, 2, len(targetDocs), "Expected 2 documents in target")
	})
}
