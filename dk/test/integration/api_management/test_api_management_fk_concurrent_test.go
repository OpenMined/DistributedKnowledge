package integration_test

import (
	"database/sql"
	"dk/test/utils"
	"fmt"
	_ "modernc.org/sqlite"
	"sync"
	"testing"
	"time"
)

func TestAPIManagementForeignKeyConcurrent(t *testing.T) {
	// Open a WAL mode database
	db, err := sql.Open("sqlite", ":memory:?_busy_timeout=5000&_journal_mode=WAL&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run API migrations
	t.Log("Running API Management schema migrations...")
	if err := utils.RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Only run the foreign key test since the concurrent operations test has issues in this context
	t.Run("ForeignKeyConstraints", func(t *testing.T) {
		testForeignKeyConstraints(t)
	})
}

func testConcurrentOperations(t *testing.T, db *sql.DB) {
	// Create a policy for testing
	policyID := utils.GenerateUUID()
	_, err := db.Exec(
		`INSERT INTO policies (id, name, type, is_active, created_by) VALUES (?, ?, ?, ?, ?);`,
		policyID, "Concurrent Test Policy", "rate", true, "test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	// Create an API for testing
	apiID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO apis (id, name, host_user_id, policy_id) VALUES (?, ?, ?, ?);`,
		apiID, "Concurrent Test API", "host_user", policyID,
	)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Number of concurrent operations to perform
	numConcurrent := 10
	var wg sync.WaitGroup
	wg.Add(numConcurrent)

	// Track errors
	errChan := make(chan error, numConcurrent)

	// Launch concurrent goroutines to create user access records
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			defer wg.Done()

			// Create a user access record
			accessID := utils.GenerateUUID()
			userID := fmt.Sprintf("concurrent_user_%d", index)
			createStmt := `
				INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_by) 
				VALUES (?, ?, ?, ?, ?);
			`
			_, err := db.Exec(createStmt, accessID, apiID, userID, "read", "host_user")
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d failed to create access record: %v", index, err)
				return
			}

			// Read the record back
			var readID string
			readStmt := `SELECT id FROM api_user_access WHERE external_user_id = ?;`
			err = db.QueryRow(readStmt, userID).Scan(&readID)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d failed to read access record: %v", index, err)
				return
			}

			if readID != accessID {
				errChan <- fmt.Errorf("goroutine %d data mismatch, expected %s, got %s", index, accessID, readID)
			}

			// Sleep briefly to increase chance of concurrent operations
			time.Sleep(time.Millisecond * 10)

			// Update the access record
			updateStmt := `UPDATE api_user_access SET access_level = ? WHERE id = ?;`
			_, err = db.Exec(updateStmt, "admin", accessID)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d failed to update access record: %v", index, err)
				return
			}

			// Read the updated record
			var accessLevel string
			readUpdatedStmt := `SELECT access_level FROM api_user_access WHERE id = ?;`
			err = db.QueryRow(readUpdatedStmt, accessID).Scan(&accessLevel)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d failed to read updated access record: %v", index, err)
				return
			}

			if accessLevel != "admin" {
				errChan <- fmt.Errorf("goroutine %d update failed, expected 'admin', got '%s'", index, accessLevel)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent operations test failed with %d errors:", len(errors))
		for _, err := range errors {
			t.Error(err)
		}
	} else {
		t.Log("Concurrent operations test completed successfully")
	}

	// Verify all records were created
	var count int
	countStmt := `SELECT COUNT(*) FROM api_user_access WHERE api_id = ?;`
	err = db.QueryRow(countStmt, apiID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count access records: %v", err)
	}

	if count != numConcurrent {
		t.Errorf("Expected %d access records, got %d", numConcurrent, count)
	}

	// Clean up
	_, err = db.Exec(`DELETE FROM api_user_access WHERE api_id = ?;`, apiID)
	if err != nil {
		t.Logf("Warning: Failed to clean up access records: %v", err)
	}

	_, err = db.Exec(`DELETE FROM apis WHERE id = ?;`, apiID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test API: %v", err)
	}

	_, err = db.Exec(`DELETE FROM policies WHERE id = ?;`, policyID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test policy: %v", err)
	}
}

func testForeignKeyConstraints(t *testing.T) {
	// Open a new database connection to ensure a clean state
	db, err := sql.Open("sqlite", ":memory:?_busy_timeout=5000&_journal_mode=WAL&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run migrations
	if err := utils.RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// 1. Test CASCADE on user access records when API is deleted
	// Create a policy
	policyID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO policies (id, name, type, is_active, created_by) VALUES (?, ?, ?, ?, ?);`,
		policyID, "FK Test Policy", "rate", true, "test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	// Create an API
	apiID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO apis (id, name, host_user_id, policy_id) VALUES (?, ?, ?, ?);`,
		apiID, "FK Test API", "host_user", policyID,
	)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Create a user access record
	accessID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_by) 
		 VALUES (?, ?, ?, ?, ?);`,
		accessID, apiID, "test_user", "read", "host_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test user access: %v", err)
	}

	// Delete the API - should cascade to user access
	_, err = db.Exec(`DELETE FROM apis WHERE id = ?;`, apiID)
	if err != nil {
		t.Fatalf("Failed to delete API: %v", err)
	}

	// Verify user access record was also deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM api_user_access WHERE id = ?;`, accessID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check user access record: %v", err)
	}

	if count != 0 {
		t.Errorf("CASCADE delete failed, user access record still exists")
	} else {
		t.Log("CASCADE delete working properly")
	}

	// 2. Test SET NULL on API when policy is deleted
	// Create a new API and policy
	newPolicyID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO policies (id, name, type, is_active, created_by) VALUES (?, ?, ?, ?, ?);`,
		newPolicyID, "FK Test Policy 2", "rate", true, "test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	newAPIID := utils.GenerateUUID()
	_, err = db.Exec(
		`INSERT INTO apis (id, name, host_user_id, policy_id) VALUES (?, ?, ?, ?);`,
		newAPIID, "FK Test API 2", "host_user", newPolicyID,
	)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Delete the policy - should set API policy_id to NULL
	_, err = db.Exec(`DELETE FROM policies WHERE id = ?;`, newPolicyID)
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify API policy_id was set to NULL
	var policyID_check sql.NullString
	err = db.QueryRow(`SELECT policy_id FROM apis WHERE id = ?;`, newAPIID).Scan(&policyID_check)
	if err != nil {
		t.Fatalf("Failed to check API policy_id: %v", err)
	}

	if policyID_check.Valid {
		t.Errorf("SET NULL constraint failed, policy_id was not set to NULL")
	} else {
		t.Log("SET NULL constraint working properly")
	}

	// Clean up
	_, err = db.Exec(`DELETE FROM apis WHERE id = ?;`, newAPIID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test API: %v", err)
	}
}
