package db

import (
	"database/sql"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"os"
	"testing"
	"time"
)

// TestRunAPIMigrations verifies that the API Management tables are created properly
func TestRunAPIMigrations(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Define the expected tables from the API Management schema
	expectedTables := []string{
		"policies",
		"apis",
		"api_requests",
		"document_associations",
		"api_user_access",
		"trackers",
		"request_required_trackers",
		"policy_rules",
		"api_usage",
		"api_usage_summary",
		"policy_changes",
		"quota_notifications",
	}

	// Check if all expected tables were created
	for _, tableName := range expectedTables {
		// Query to check if table exists
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?;"
		var name string
		err := db.QueryRow(query, tableName).Scan(&name)

		if err != nil {
			if err == sql.ErrNoRows {
				t.Errorf("Table %s was not created", tableName)
			} else {
				t.Errorf("Error checking for table %s: %v", tableName, err)
			}
		} else if name != tableName {
			t.Errorf("Expected table name %s, got %s", tableName, name)
		}
	}
}

// TestTableColumnDefinitions checks that tables have the expected columns
func TestTableColumnDefinitions(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Define test cases for column checks
	testCases := []struct {
		tableName       string
		expectedColumns []string
	}{
		{
			tableName: "apis",
			expectedColumns: []string{
				"id", "name", "description", "created_at", "updated_at",
				"is_active", "api_key", "host_user_id", "policy_id",
				"is_deprecated", "deprecation_date", "deprecation_message",
			},
		},
		{
			tableName: "api_requests",
			expectedColumns: []string{
				"id", "api_name", "description", "submitted_date", "status",
				"requester_id", "denial_reason", "denied_date", "approved_date",
				"submission_count", "previous_request_id",
			},
		},
		{
			tableName: "policies",
			expectedColumns: []string{
				"id", "name", "description", "type", "is_active",
				"created_at", "updated_at", "created_by",
			},
		},
		// Add more tables as needed
	}

	// Run the column checks for each table
	for _, tc := range testCases {
		t.Run("Table_"+tc.tableName, func(t *testing.T) {
			// Query to get column info for the table
			rows, err := db.Query("PRAGMA table_info(" + tc.tableName + ")")
			if err != nil {
				t.Fatalf("Failed to get column info for table %s: %v", tc.tableName, err)
			}
			defer rows.Close()

			// Collect actual columns
			var actualColumns []string
			for rows.Next() {
				var cid int
				var name, dataType string
				var notNull, pk int
				var dfltValue interface{}
				if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
					t.Fatalf("Failed to scan column data: %v", err)
				}
				actualColumns = append(actualColumns, name)
			}

			// Check if all expected columns exist
			for _, expectedCol := range tc.expectedColumns {
				found := false
				for _, actualCol := range actualColumns {
					if expectedCol == actualCol {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected column %s not found in table %s", expectedCol, tc.tableName)
				}
			}
		})
	}
}

// TestForeignKeyRelationships verifies the foreign key relationships between tables
func TestForeignKeyRelationships(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Define expected foreign key relationships
	relationships := []struct {
		tableName    string
		foreignKeys  []string
		parentTables []string
	}{
		{
			tableName:    "apis",
			foreignKeys:  []string{"policy_id"},
			parentTables: []string{"policies"},
		},
		{
			tableName:    "api_requests",
			foreignKeys:  []string{"previous_request_id"},
			parentTables: []string{"api_requests"},
		},
		{
			tableName:    "api_user_access",
			foreignKeys:  []string{"api_id"},
			parentTables: []string{"apis"},
		},
		{
			tableName:    "request_required_trackers",
			foreignKeys:  []string{"request_id", "tracker_id"},
			parentTables: []string{"api_requests", "trackers"},
		},
		{
			tableName:    "policy_rules",
			foreignKeys:  []string{"policy_id"},
			parentTables: []string{"policies"},
		},
		{
			tableName:    "api_usage",
			foreignKeys:  []string{"api_id"},
			parentTables: []string{"apis"},
		},
		{
			tableName:    "api_usage_summary",
			foreignKeys:  []string{"api_id"},
			parentTables: []string{"apis"},
		},
		{
			tableName:    "policy_changes",
			foreignKeys:  []string{"api_id", "old_policy_id", "new_policy_id"},
			parentTables: []string{"apis", "policies", "policies"},
		},
		{
			tableName:    "quota_notifications",
			foreignKeys:  []string{"api_id"},
			parentTables: []string{"apis"},
		},
	}

	// Check foreign key relationships
	for _, rel := range relationships {
		t.Run("ForeignKey_"+rel.tableName, func(t *testing.T) {
			// Get foreign key info for the table
			rows, err := db.Query("PRAGMA foreign_key_list(" + rel.tableName + ")")
			if err != nil {
				t.Fatalf("Failed to get foreign key info for table %s: %v", rel.tableName, err)
			}
			defer rows.Close()

			// Track which foreign keys we've found
			foundFKs := make(map[string]bool)
			for _, fk := range rel.foreignKeys {
				foundFKs[fk] = false
			}

			// Check each foreign key
			for rows.Next() {
				var id, seq int
				var table, from, to string
				var onUpdate, onDelete, match string
				if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
					t.Fatalf("Failed to scan foreign key data: %v", err)
				}

				// Find this FK in our expectations
				for i, fk := range rel.foreignKeys {
					if fk == from {
						foundFKs[fk] = true
						if rel.parentTables[i] != table {
							t.Errorf("Foreign key %s in table %s references table %s, expected %s",
								from, rel.tableName, table, rel.parentTables[i])
						}
					}
				}
			}

			// Check if all expected foreign keys were found
			for fk, found := range foundFKs {
				if !found {
					t.Errorf("Expected foreign key %s not found in table %s", fk, rel.tableName)
				}
			}
		})
	}
}

// Helper function to insert test data - will implement in the CRUD tests
func insertTestData(t *testing.T, db *sql.DB) (policyID, apiID, trackerID, requestID string) {
	// Generate UUIDs for our test records
	policyID = uuid.New().String()
	apiID = uuid.New().String()
	trackerID = uuid.New().String()
	requestID = uuid.New().String()

	// Insert test policy
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		policyID, "Test Policy", "A policy for testing", "free", true,
		time.Now(), time.Now(), "test_user")
	if err != nil {
		t.Fatalf("Failed to insert test policy: %v", err)
	}

	// Insert test API using the policy
	_, err = db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id, policy_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		apiID, "Test API", "An API for testing", true, "test_key_123", "test_host", policyID)
	if err != nil {
		t.Fatalf("Failed to insert test API: %v", err)
	}

	// Insert test tracker
	_, err = db.Exec(`
		INSERT INTO trackers (id, name, description, is_active)
		VALUES (?, ?, ?, ?)`,
		trackerID, "Test Tracker", "A tracker for testing", true)
	if err != nil {
		t.Fatalf("Failed to insert test tracker: %v", err)
	}

	// Insert test API request
	_, err = db.Exec(`
		INSERT INTO api_requests (id, api_name, description, status, requester_id)
		VALUES (?, ?, ?, ?, ?)`,
		requestID, "Test API", "A request for testing", "pending", "test_requester")
	if err != nil {
		t.Fatalf("Failed to insert test API request: %v", err)
	}

	return policyID, apiID, trackerID, requestID
}

// TestForeignKeyConstraints verifies that foreign key constraints are enforced
func TestForeignKeyConstraints(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Insert test data
	policyID, apiID, trackerID, requestID := insertTestData(t, db)

	// Test cases for constraint enforcement
	testCases := []struct {
		name        string
		query       string
		args        []interface{}
		shouldFail  bool
		description string
	}{
		{
			name: "Invalid policy reference",
			query: `INSERT INTO apis (id, name, host_user_id, policy_id)
                    VALUES (?, ?, ?, ?)`,
			args:        []interface{}{uuid.New().String(), "Invalid API", "test_host", "non_existent_policy"},
			shouldFail:  true,
			description: "Should fail due to non-existent policy_id foreign key",
		},
		{
			name: "Invalid API reference",
			query: `INSERT INTO api_user_access (id, api_id, external_user_id, access_level)
                    VALUES (?, ?, ?, ?)`,
			args:        []interface{}{uuid.New().String(), "non_existent_api", "test_user", "read"},
			shouldFail:  true,
			description: "Should fail due to non-existent api_id foreign key",
		},
		{
			name: "Valid API user access",
			query: `INSERT INTO api_user_access (id, api_id, external_user_id, access_level)
                    VALUES (?, ?, ?, ?)`,
			args:        []interface{}{uuid.New().String(), apiID, "test_user", "read"},
			shouldFail:  false,
			description: "Should succeed with valid api_id",
		},
		{
			name: "Valid policy rule",
			query: `INSERT INTO policy_rules (id, policy_id, rule_type, action, priority)
                    VALUES (?, ?, ?, ?, ?)`,
			args:        []interface{}{uuid.New().String(), policyID, "token", "block", 100},
			shouldFail:  false,
			description: "Should succeed with valid policy_id",
		},
		{
			name: "Valid tracker association",
			query: `INSERT INTO request_required_trackers (id, request_id, tracker_id)
                    VALUES (?, ?, ?)`,
			args:        []interface{}{uuid.New().String(), requestID, trackerID},
			shouldFail:  false,
			description: "Should succeed with valid request_id and tracker_id",
		},
	}

	// Make sure foreign keys are enabled
	_, err := db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign key constraints: %v", err)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign key status: %v", err)
	}
	if fkEnabled != 1 {
		t.Fatalf("Foreign key constraints are not enabled")
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Enable foreign keys for each test
			_, _ = db.Exec("PRAGMA foreign_keys = ON;")

			_, err := db.Exec(tc.query, tc.args...)

			if tc.shouldFail && err == nil {
				t.Errorf("Expected constraint violation, but got success: %s", tc.description)
			} else if !tc.shouldFail && err != nil {
				t.Errorf("Expected success, but got error: %v - %s", err, tc.description)
			}
		})
	}
}
