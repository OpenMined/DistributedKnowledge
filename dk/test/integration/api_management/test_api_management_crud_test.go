package integration_test

import (
	"database/sql"
	"dk/test/utils"
	_ "modernc.org/sqlite"
	"testing"
	"time"
)

func TestAPIManagementCRUD(t *testing.T) {
	// Open an in-memory database for testing
	db, err := sql.Open("sqlite", ":memory:?_busy_timeout=5000&_journal_mode=DELETE&cache=shared")
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

	// Test Policy CRUD operations
	t.Run("PolicyCRUD", func(t *testing.T) {
		testPolicyCRUD(t, db)
	})

	// Test API CRUD operations
	t.Run("APICRUD", func(t *testing.T) {
		testAPICRUD(t, db)
	})

	// Test API Request CRUD operations
	t.Run("APIRequestCRUD", func(t *testing.T) {
		testAPIRequestCRUD(t, db)
	})

	// Test Document Association CRUD operations
	t.Run("DocumentAssociationCRUD", func(t *testing.T) {
		testDocumentAssociationCRUD(t, db)
	})

	// Test API User Access CRUD operations
	t.Run("APIUserAccessCRUD", func(t *testing.T) {
		testAPIUserAccessCRUD(t, db)
	})

	// Test Tracker CRUD operations
	t.Run("TrackerCRUD", func(t *testing.T) {
		testTrackerCRUD(t, db)
	})
}

func testPolicyCRUD(t *testing.T, db *sql.DB) {
	// Create a policy
	policyID := utils.GenerateUUID()
	createStmt := `
		INSERT INTO policies (id, name, description, type, is_active, created_by) 
		VALUES (?, ?, ?, ?, ?, ?);
	`
	_, err := db.Exec(createStmt, policyID, "Test Policy", "Policy for testing", "rate", true, "test_user")
	if err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	// Read the policy
	var (
		id          string
		name        string
		description string
		policyType  string
		isActive    bool
		createdBy   string
	)
	readStmt := `SELECT id, name, description, type, is_active, created_by FROM policies WHERE id = ?;`
	err = db.QueryRow(readStmt, policyID).Scan(&id, &name, &description, &policyType, &isActive, &createdBy)
	if err != nil {
		t.Fatalf("Failed to read test policy: %v", err)
	}

	// Verify data
	if id != policyID || name != "Test Policy" || description != "Policy for testing" ||
		policyType != "rate" || !isActive || createdBy != "test_user" {
		t.Errorf("Policy data mismatch, got: %s, %s, %s, %s, %v, %s",
			id, name, description, policyType, isActive, createdBy)
	} else {
		t.Log("Policy created and read successfully")
	}

	// Update the policy
	updateStmt := `UPDATE policies SET name = ?, description = ? WHERE id = ?;`
	_, err = db.Exec(updateStmt, "Updated Policy", "Updated description", policyID)
	if err != nil {
		t.Fatalf("Failed to update test policy: %v", err)
	}

	// Verify update
	var updatedName, updatedDesc string
	readUpdatedStmt := `SELECT name, description FROM policies WHERE id = ?;`
	err = db.QueryRow(readUpdatedStmt, policyID).Scan(&updatedName, &updatedDesc)
	if err != nil {
		t.Fatalf("Failed to read updated policy: %v", err)
	}

	if updatedName != "Updated Policy" || updatedDesc != "Updated description" {
		t.Errorf("Policy update failed, got: %s, %s", updatedName, updatedDesc)
	} else {
		t.Log("Policy updated successfully")
	}

	// Delete the policy
	deleteStmt := `DELETE FROM policies WHERE id = ?;`
	_, err = db.Exec(deleteStmt, policyID)
	if err != nil {
		t.Fatalf("Failed to delete test policy: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM policies WHERE id = ?;`
	err = db.QueryRow(countStmt, policyID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify policy deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Policy deletion failed, policy still exists")
	} else {
		t.Log("Policy deleted successfully")
	}
}

func testAPICRUD(t *testing.T, db *sql.DB) {
	// Create a policy first for foreign key relationship
	policyID := utils.GenerateUUID()
	_, err := db.Exec(
		`INSERT INTO policies (id, name, type, is_active, created_by) VALUES (?, ?, ?, ?, ?);`,
		policyID, "API Test Policy", "rate", true, "test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test policy for API: %v", err)
	}

	// Create an API
	apiID := utils.GenerateUUID()
	apiKey := "api_key_" + utils.GenerateUUID()
	createStmt := `
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id, policy_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	_, err = db.Exec(createStmt, apiID, "Test API", "API for testing", true, apiKey, "host_user", policyID)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Read the API
	var (
		id              string
		name            string
		desc            string
		isActive        bool
		fetchedKey      string
		hostUserID      string
		fetchedPolicyID sql.NullString
	)
	readStmt := `SELECT id, name, description, is_active, api_key, host_user_id, policy_id FROM apis WHERE id = ?;`
	err = db.QueryRow(readStmt, apiID).Scan(&id, &name, &desc, &isActive, &fetchedKey, &hostUserID, &fetchedPolicyID)
	if err != nil {
		t.Fatalf("Failed to read test API: %v", err)
	}

	// Verify data
	if id != apiID || name != "Test API" || desc != "API for testing" ||
		!isActive || fetchedKey != apiKey || hostUserID != "host_user" || fetchedPolicyID.String != policyID {
		t.Errorf("API data mismatch")
	} else {
		t.Log("API created and read successfully")
	}

	// Update the API
	updateStmt := `UPDATE apis SET name = ?, description = ? WHERE id = ?;`
	_, err = db.Exec(updateStmt, "Updated API", "Updated API description", apiID)
	if err != nil {
		t.Fatalf("Failed to update test API: %v", err)
	}

	// Verify update
	var updatedName, updatedDesc string
	readUpdatedStmt := `SELECT name, description FROM apis WHERE id = ?;`
	err = db.QueryRow(readUpdatedStmt, apiID).Scan(&updatedName, &updatedDesc)
	if err != nil {
		t.Fatalf("Failed to read updated API: %v", err)
	}

	if updatedName != "Updated API" || updatedDesc != "Updated API description" {
		t.Errorf("API update failed, got: %s, %s", updatedName, updatedDesc)
	} else {
		t.Log("API updated successfully")
	}

	// Delete the API
	deleteStmt := `DELETE FROM apis WHERE id = ?;`
	_, err = db.Exec(deleteStmt, apiID)
	if err != nil {
		t.Fatalf("Failed to delete test API: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM apis WHERE id = ?;`
	err = db.QueryRow(countStmt, apiID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify API deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("API deletion failed, API still exists")
	} else {
		t.Log("API deleted successfully")
	}

	// Clean up policy
	_, err = db.Exec(`DELETE FROM policies WHERE id = ?;`, policyID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test policy: %v", err)
	}
}

func testAPIRequestCRUD(t *testing.T, db *sql.DB) {
	// Create an API request
	requestID := utils.GenerateUUID()
	createStmt := `
		INSERT INTO api_requests (id, api_name, description, status, requester_id) 
		VALUES (?, ?, ?, ?, ?);
	`
	_, err := db.Exec(createStmt, requestID, "Test API", "Request for testing", "pending", "test_requester")
	if err != nil {
		t.Fatalf("Failed to create test API request: %v", err)
	}

	// Read the request
	var (
		id          string
		apiName     string
		description string
		status      string
		requesterID string
	)
	readStmt := `SELECT id, api_name, description, status, requester_id FROM api_requests WHERE id = ?;`
	err = db.QueryRow(readStmt, requestID).Scan(&id, &apiName, &description, &status, &requesterID)
	if err != nil {
		t.Fatalf("Failed to read test API request: %v", err)
	}

	// Verify data
	if id != requestID || apiName != "Test API" || description != "Request for testing" ||
		status != "pending" || requesterID != "test_requester" {
		t.Errorf("API Request data mismatch")
	} else {
		t.Log("API Request created and read successfully")
	}

	// Update the request
	updateStmt := `UPDATE api_requests SET status = ?, denial_reason = ? WHERE id = ?;`
	_, err = db.Exec(updateStmt, "denied", "Not approved for testing", requestID)
	if err != nil {
		t.Fatalf("Failed to update test API request: %v", err)
	}

	// Verify update
	var updatedStatus, denialReason string
	readUpdatedStmt := `SELECT status, denial_reason FROM api_requests WHERE id = ?;`
	err = db.QueryRow(readUpdatedStmt, requestID).Scan(&updatedStatus, &denialReason)
	if err != nil {
		t.Fatalf("Failed to read updated API request: %v", err)
	}

	if updatedStatus != "denied" || denialReason != "Not approved for testing" {
		t.Errorf("API Request update failed, got: %s, %s", updatedStatus, denialReason)
	} else {
		t.Log("API Request updated successfully")
	}

	// Delete the request
	deleteStmt := `DELETE FROM api_requests WHERE id = ?;`
	_, err = db.Exec(deleteStmt, requestID)
	if err != nil {
		t.Fatalf("Failed to delete test API request: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM api_requests WHERE id = ?;`
	err = db.QueryRow(countStmt, requestID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify API request deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("API Request deletion failed, request still exists")
	} else {
		t.Log("API Request deleted successfully")
	}
}

func testDocumentAssociationCRUD(t *testing.T, db *sql.DB) {
	// Create an API first for the association
	apiID := utils.GenerateUUID()
	_, err := db.Exec(
		`INSERT INTO apis (id, name, host_user_id) VALUES (?, ?, ?);`,
		apiID, "Doc Test API", "host_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test API for document association: %v", err)
	}

	// Create a document association
	associationID := utils.GenerateUUID()
	docFilename := "test_document.md"
	createStmt := `
		INSERT INTO document_associations (id, document_filename, entity_id, entity_type) 
		VALUES (?, ?, ?, ?);
	`
	_, err = db.Exec(createStmt, associationID, docFilename, apiID, "api")
	if err != nil {
		t.Fatalf("Failed to create test document association: %v", err)
	}

	// Read the association
	var (
		id         string
		filename   string
		entityID   string
		entityType string
	)
	readStmt := `SELECT id, document_filename, entity_id, entity_type 
	             FROM document_associations WHERE id = ?;`
	err = db.QueryRow(readStmt, associationID).Scan(&id, &filename, &entityID, &entityType)
	if err != nil {
		t.Fatalf("Failed to read test document association: %v", err)
	}

	// Verify data
	if id != associationID || filename != docFilename || entityID != apiID || entityType != "api" {
		t.Errorf("Document association data mismatch")
	} else {
		t.Log("Document association created and read successfully")
	}

	// Test uniqueness constraint
	_, err = db.Exec(createStmt, utils.GenerateUUID(), docFilename, apiID, "api")
	if err == nil {
		t.Errorf("Uniqueness constraint failed, duplicate document association created")
	} else {
		t.Log("Uniqueness constraint working properly")
	}

	// Delete the association
	deleteStmt := `DELETE FROM document_associations WHERE id = ?;`
	_, err = db.Exec(deleteStmt, associationID)
	if err != nil {
		t.Fatalf("Failed to delete test document association: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM document_associations WHERE id = ?;`
	err = db.QueryRow(countStmt, associationID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify document association deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Document association deletion failed, association still exists")
	} else {
		t.Log("Document association deleted successfully")
	}

	// Clean up API
	_, err = db.Exec(`DELETE FROM apis WHERE id = ?;`, apiID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test API: %v", err)
	}
}

func testAPIUserAccessCRUD(t *testing.T, db *sql.DB) {
	// Create an API first for the access record
	apiID := utils.GenerateUUID()
	_, err := db.Exec(
		`INSERT INTO apis (id, name, host_user_id) VALUES (?, ?, ?);`,
		apiID, "Access Test API", "host_user",
	)
	if err != nil {
		t.Fatalf("Failed to create test API for user access: %v", err)
	}

	// Create a user access record
	accessID := utils.GenerateUUID()
	userID := "external_user_" + utils.GenerateUUID()[0:8]
	createStmt := `
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_by) 
		VALUES (?, ?, ?, ?, ?);
	`
	_, err = db.Exec(createStmt, accessID, apiID, userID, "read", "host_user")
	if err != nil {
		t.Fatalf("Failed to create test user access record: %v", err)
	}

	// Read the access record
	var (
		id             string
		fetchedAPIID   string
		externalUserID string
		accessLevel    string
		grantedBy      string
		isActive       bool
	)
	readStmt := `SELECT id, api_id, external_user_id, access_level, granted_by, is_active 
	             FROM api_user_access WHERE id = ?;`
	err = db.QueryRow(readStmt, accessID).Scan(&id, &fetchedAPIID, &externalUserID, &accessLevel, &grantedBy, &isActive)
	if err != nil {
		t.Fatalf("Failed to read test user access record: %v", err)
	}

	// Verify data
	if id != accessID || fetchedAPIID != apiID || externalUserID != userID ||
		accessLevel != "read" || grantedBy != "host_user" || !isActive {
		t.Errorf("User access data mismatch")
	} else {
		t.Log("User access record created and read successfully")
	}

	// Update the access record
	updateStmt := `UPDATE api_user_access SET access_level = ? WHERE id = ?;`
	_, err = db.Exec(updateStmt, "admin", accessID)
	if err != nil {
		t.Fatalf("Failed to update test user access record: %v", err)
	}

	// Verify update
	var updatedAccessLevel string
	readUpdatedStmt := `SELECT access_level FROM api_user_access WHERE id = ?;`
	err = db.QueryRow(readUpdatedStmt, accessID).Scan(&updatedAccessLevel)
	if err != nil {
		t.Fatalf("Failed to read updated user access record: %v", err)
	}

	if updatedAccessLevel != "admin" {
		t.Errorf("User access update failed, got: %s", updatedAccessLevel)
	} else {
		t.Log("User access record updated successfully")
	}

	// Test the access level constraint
	invalidUpdateStmt := `UPDATE api_user_access SET access_level = ? WHERE id = ?;`
	_, err = db.Exec(invalidUpdateStmt, "invalid_level", accessID)
	if err == nil {
		t.Errorf("Access level constraint failed, invalid access level accepted")
	} else {
		t.Log("Access level constraint working properly")
	}

	// Revoke access
	revokeStmt := `UPDATE api_user_access SET is_active = ?, revoked_at = ? WHERE id = ?;`
	_, err = db.Exec(revokeStmt, false, time.Now(), accessID)
	if err != nil {
		t.Fatalf("Failed to revoke user access: %v", err)
	}

	// Verify revocation
	var isStillActive bool
	var revokedAt sql.NullTime
	readRevokedStmt := `SELECT is_active, revoked_at FROM api_user_access WHERE id = ?;`
	err = db.QueryRow(readRevokedStmt, accessID).Scan(&isStillActive, &revokedAt)
	if err != nil {
		t.Fatalf("Failed to read revoked user access record: %v", err)
	}

	if isStillActive || !revokedAt.Valid {
		t.Errorf("User access revocation failed")
	} else {
		t.Log("User access revoked successfully")
	}

	// Delete the access record
	deleteStmt := `DELETE FROM api_user_access WHERE id = ?;`
	_, err = db.Exec(deleteStmt, accessID)
	if err != nil {
		t.Fatalf("Failed to delete test user access record: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM api_user_access WHERE id = ?;`
	err = db.QueryRow(countStmt, accessID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify user access record deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("User access record deletion failed, record still exists")
	} else {
		t.Log("User access record deleted successfully")
	}

	// Clean up API
	_, err = db.Exec(`DELETE FROM apis WHERE id = ?;`, apiID)
	if err != nil {
		t.Logf("Warning: Failed to clean up test API: %v", err)
	}
}

func testTrackerCRUD(t *testing.T, db *sql.DB) {
	// Create a tracker
	trackerID := utils.GenerateUUID()
	createStmt := `
		INSERT INTO trackers (id, name, description, is_active) 
		VALUES (?, ?, ?, ?);
	`
	_, err := db.Exec(createStmt, trackerID, "Test Tracker", "Tracker for testing", true)
	if err != nil {
		t.Fatalf("Failed to create test tracker: %v", err)
	}

	// Read the tracker
	var (
		id          string
		name        string
		description string
		isActive    bool
	)
	readStmt := `SELECT id, name, description, is_active FROM trackers WHERE id = ?;`
	err = db.QueryRow(readStmt, trackerID).Scan(&id, &name, &description, &isActive)
	if err != nil {
		t.Fatalf("Failed to read test tracker: %v", err)
	}

	// Verify data
	if id != trackerID || name != "Test Tracker" || description != "Tracker for testing" || !isActive {
		t.Errorf("Tracker data mismatch")
	} else {
		t.Log("Tracker created and read successfully")
	}

	// Update the tracker
	updateStmt := `UPDATE trackers SET name = ?, description = ? WHERE id = ?;`
	_, err = db.Exec(updateStmt, "Updated Tracker", "Updated tracker description", trackerID)
	if err != nil {
		t.Fatalf("Failed to update test tracker: %v", err)
	}

	// Verify update
	var updatedName, updatedDesc string
	readUpdatedStmt := `SELECT name, description FROM trackers WHERE id = ?;`
	err = db.QueryRow(readUpdatedStmt, trackerID).Scan(&updatedName, &updatedDesc)
	if err != nil {
		t.Fatalf("Failed to read updated tracker: %v", err)
	}

	if updatedName != "Updated Tracker" || updatedDesc != "Updated tracker description" {
		t.Errorf("Tracker update failed, got: %s, %s", updatedName, updatedDesc)
	} else {
		t.Log("Tracker updated successfully")
	}

	// Delete the tracker
	deleteStmt := `DELETE FROM trackers WHERE id = ?;`
	_, err = db.Exec(deleteStmt, trackerID)
	if err != nil {
		t.Fatalf("Failed to delete test tracker: %v", err)
	}

	// Verify deletion
	var count int
	countStmt := `SELECT COUNT(*) FROM trackers WHERE id = ?;`
	err = db.QueryRow(countStmt, trackerID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify tracker deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Tracker deletion failed, tracker still exists")
	} else {
		t.Log("Tracker deleted successfully")
	}
}
