package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestFixedTransactionHandling tests the transaction functionality with
// a proper handling of SQLite locking behavior.
func TestFixedTransactionHandling(t *testing.T) {
	// Create a dedicated test database for this test
	testDBPath := fmt.Sprintf("/tmp/test_db_%s.db", uuid.New().String())

	// Open the main database connection
	mainDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open main database: %v", err)
	}
	defer mainDB.Close()

	// Initialize database schema
	_, err = mainDB.Exec(`
		CREATE TABLE document_associations (
			id TEXT PRIMARY KEY,
			document_filename TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
		
		CREATE TABLE apis (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			host_user_id TEXT NOT NULL
		);
		
		CREATE TABLE api_requests (
			id TEXT PRIMARY KEY,
			api_name TEXT NOT NULL,
			status TEXT NOT NULL,
			requester_id TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create test entity IDs
	apiID := uuid.New().String()
	requestID := uuid.New().String()

	// Insert test entities
	_, err = mainDB.Exec(
		"INSERT INTO apis (id, name, host_user_id) VALUES (?, ?, ?)",
		apiID, "Test API", "test_user")
	if err != nil {
		t.Fatalf("Failed to insert API: %v", err)
	}

	_, err = mainDB.Exec(
		"INSERT INTO api_requests (id, api_name, status, requester_id) VALUES (?, ?, ?, ?)",
		requestID, "Test API", "pending", "test_user")
	if err != nil {
		t.Fatalf("Failed to insert API request: %v", err)
	}

	// Create a unique document filename for this test
	docFilename := fmt.Sprintf("test_doc_%s.pdf", uuid.New().String())

	// Insert test documents
	for i := 0; i < 2; i++ {
		entityID := apiID
		entityType := "api"
		if i == 1 {
			entityID = requestID
			entityType = "request"
		}

		_, err = mainDB.Exec(
			"INSERT INTO document_associations (id, document_filename, entity_id, entity_type, created_at) VALUES (?, ?, ?, ?, ?)",
			uuid.New().String(), docFilename, entityID, entityType, time.Now())
		if err != nil {
			t.Fatalf("Failed to insert document %d: %v", i, err)
		}
	}

	// Verify documents exist
	var count int
	err = mainDB.QueryRow("SELECT COUNT(*) FROM document_associations WHERE document_filename = ?", docFilename).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}
	assert.Equal(t, 2, count, "Expected 2 documents before transaction")

	// Begin a transaction to delete documents
	tx, err := mainDB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Delete documents within transaction but don't commit yet
	_, err = tx.Exec("DELETE FROM document_associations WHERE document_filename = ?", docFilename)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to delete documents in transaction: %v", err)
	}

	// Open a SEPARATE connection to verify documents still exist
	// Use PRAGMA read_uncommitted=true to avoid locking issues
	verifyDB, err := sql.Open("sqlite", testDBPath+"?_pragma=read_uncommitted(true)")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to open verification database: %v", err)
	}
	defer verifyDB.Close()

	// Verify documents still exist from separate connection
	err = verifyDB.QueryRow("SELECT COUNT(*) FROM document_associations WHERE document_filename = ?", docFilename).Scan(&count)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to count documents during transaction: %v", err)
	}

	// With SQLite's default isolation, the documents should still be visible
	// unless we've specified read_uncommitted
	assert.Equal(t, 2, count, "Expected documents to still exist during transaction")

	// Now commit the transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify documents are gone
	err = mainDB.QueryRow("SELECT COUNT(*) FROM document_associations WHERE document_filename = ?", docFilename).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count documents after commit: %v", err)
	}
	assert.Equal(t, 0, count, "Expected no documents after commit")
}
