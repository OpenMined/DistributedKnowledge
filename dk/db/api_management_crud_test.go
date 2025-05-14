package db

import (
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"os"
	"testing"
	"time"
)

// TestPolicyCRUD tests the CRUD operations for the policies table
func TestPolicyCRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Test Policy CREATE
	policyID := uuid.New().String()
	now := time.Now()

	// Insert a policy
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policyID, "Test Policy", "Policy for testing", "token", true, now, now, "test_user")

	if err != nil {
		t.Fatalf("Failed to insert test policy: %v", err)
	}

	// Test Policy READ
	var retrievedPolicy struct {
		ID          string
		Name        string
		Description string
		Type        string
		IsActive    bool
		CreatedBy   string
	}

	err = db.QueryRow(`
		SELECT id, name, description, type, is_active, created_by
		FROM policies
		WHERE id = ?
	`, policyID).Scan(
		&retrievedPolicy.ID,
		&retrievedPolicy.Name,
		&retrievedPolicy.Description,
		&retrievedPolicy.Type,
		&retrievedPolicy.IsActive,
		&retrievedPolicy.CreatedBy,
	)

	if err != nil {
		t.Fatalf("Failed to read policy: %v", err)
	}

	// Verify retrieved data
	if retrievedPolicy.ID != policyID {
		t.Errorf("Expected policy ID %s, got %s", policyID, retrievedPolicy.ID)
	}
	if retrievedPolicy.Name != "Test Policy" {
		t.Errorf("Expected policy name 'Test Policy', got '%s'", retrievedPolicy.Name)
	}
	if retrievedPolicy.Type != "token" {
		t.Errorf("Expected policy type 'token', got '%s'", retrievedPolicy.Type)
	}
	if retrievedPolicy.CreatedBy != "test_user" {
		t.Errorf("Expected created_by 'test_user', got '%s'", retrievedPolicy.CreatedBy)
	}

	// Test Policy UPDATE
	_, err = db.Exec(`
		UPDATE policies
		SET name = ?, description = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, "Updated Policy", "Updated description", false, time.Now(), policyID)

	if err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// Verify update
	var updatedName, updatedDescription string
	var updatedIsActive bool

	err = db.QueryRow(`
		SELECT name, description, is_active
		FROM policies
		WHERE id = ?
	`, policyID).Scan(&updatedName, &updatedDescription, &updatedIsActive)

	if err != nil {
		t.Fatalf("Failed to read updated policy: %v", err)
	}

	if updatedName != "Updated Policy" {
		t.Errorf("Update failed: expected name 'Updated Policy', got '%s'", updatedName)
	}
	if updatedDescription != "Updated description" {
		t.Errorf("Update failed: expected description 'Updated description', got '%s'", updatedDescription)
	}
	if updatedIsActive != false {
		t.Errorf("Update failed: expected is_active 'false', got '%v'", updatedIsActive)
	}

	// Test Policy DELETE
	_, err = db.Exec(`DELETE FROM policies WHERE id = ?`, policyID)
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM policies WHERE id = ?`, policyID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check policy deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Delete failed: policy still exists with ID %s", policyID)
	}
}

// TestAPICRUD tests the CRUD operations for the APIs table
func TestAPICRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// First create a policy to reference
	policyID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policyID, "API Test Policy", "Policy for API testing", "free", true, time.Now(), time.Now(), "test_user")

	if err != nil {
		t.Fatalf("Failed to insert policy for API test: %v", err)
	}

	// Test API CREATE
	apiID := uuid.New().String()
	now := time.Now()
	apiKey := "test_api_key_123"

	_, err = db.Exec(`
		INSERT INTO apis (id, name, description, created_at, updated_at, is_active, api_key, host_user_id, policy_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", now, now, true, apiKey, "test_host", policyID)

	if err != nil {
		t.Fatalf("Failed to insert test API: %v", err)
	}

	// Test API READ
	var retrievedAPI struct {
		ID          string
		Name        string
		Description string
		IsActive    bool
		APIKey      string
		HostUserID  string
		PolicyID    string
	}

	err = db.QueryRow(`
		SELECT id, name, description, is_active, api_key, host_user_id, policy_id
		FROM apis
		WHERE id = ?
	`, apiID).Scan(
		&retrievedAPI.ID,
		&retrievedAPI.Name,
		&retrievedAPI.Description,
		&retrievedAPI.IsActive,
		&retrievedAPI.APIKey,
		&retrievedAPI.HostUserID,
		&retrievedAPI.PolicyID,
	)

	if err != nil {
		t.Fatalf("Failed to read API: %v", err)
	}

	// Verify retrieved data
	if retrievedAPI.ID != apiID {
		t.Errorf("Expected API ID %s, got %s", apiID, retrievedAPI.ID)
	}
	if retrievedAPI.Name != "Test API" {
		t.Errorf("Expected API name 'Test API', got '%s'", retrievedAPI.Name)
	}
	if retrievedAPI.APIKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, retrievedAPI.APIKey)
	}
	if retrievedAPI.PolicyID != policyID {
		t.Errorf("Expected policy ID '%s', got '%s'", policyID, retrievedAPI.PolicyID)
	}

	// Test API UPDATE
	_, err = db.Exec(`
		UPDATE apis
		SET name = ?, description = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, "Updated API", "Updated API description", false, time.Now(), apiID)

	if err != nil {
		t.Fatalf("Failed to update API: %v", err)
	}

	// Verify update
	var updatedName, updatedDescription string
	var updatedIsActive bool

	err = db.QueryRow(`
		SELECT name, description, is_active
		FROM apis
		WHERE id = ?
	`, apiID).Scan(&updatedName, &updatedDescription, &updatedIsActive)

	if err != nil {
		t.Fatalf("Failed to read updated API: %v", err)
	}

	if updatedName != "Updated API" {
		t.Errorf("Update failed: expected name 'Updated API', got '%s'", updatedName)
	}
	if updatedDescription != "Updated API description" {
		t.Errorf("Update failed: expected description 'Updated API description', got '%s'", updatedDescription)
	}
	if updatedIsActive != false {
		t.Errorf("Update failed: expected is_active 'false', got '%v'", updatedIsActive)
	}

	// Test API DELETE
	_, err = db.Exec(`DELETE FROM apis WHERE id = ?`, apiID)
	if err != nil {
		t.Fatalf("Failed to delete API: %v", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM apis WHERE id = ?`, apiID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check API deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Delete failed: API still exists with ID %s", apiID)
	}
}

// TestAPIRequestCRUD tests the CRUD operations for the api_requests table
func TestAPIRequestCRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Test API Request CREATE
	requestID := uuid.New().String()
	now := time.Now()

	_, err := db.Exec(`
		INSERT INTO api_requests (id, api_name, description, submitted_date, status, requester_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, requestID, "Test API", "Request for testing", now, "pending", "test_requester")

	if err != nil {
		t.Fatalf("Failed to insert test API request: %v", err)
	}

	// Test API Request READ
	var retrievedRequest struct {
		ID          string
		APIName     string
		Description string
		Status      string
		RequesterID string
	}

	err = db.QueryRow(`
		SELECT id, api_name, description, status, requester_id
		FROM api_requests
		WHERE id = ?
	`, requestID).Scan(
		&retrievedRequest.ID,
		&retrievedRequest.APIName,
		&retrievedRequest.Description,
		&retrievedRequest.Status,
		&retrievedRequest.RequesterID,
	)

	if err != nil {
		t.Fatalf("Failed to read API request: %v", err)
	}

	// Verify retrieved data
	if retrievedRequest.ID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, retrievedRequest.ID)
	}
	if retrievedRequest.APIName != "Test API" {
		t.Errorf("Expected API name 'Test API', got '%s'", retrievedRequest.APIName)
	}
	if retrievedRequest.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", retrievedRequest.Status)
	}

	// Test API Request UPDATE - for example, approve the request
	approvalTime := time.Now()
	_, err = db.Exec(`
		UPDATE api_requests
		SET status = ?, approved_date = ?
		WHERE id = ?
	`, "approved", approvalTime, requestID)

	if err != nil {
		t.Fatalf("Failed to update API request: %v", err)
	}

	// Verify update
	var updatedStatus string
	var updatedApprovalDate time.Time

	err = db.QueryRow(`
		SELECT status, approved_date
		FROM api_requests
		WHERE id = ?
	`, requestID).Scan(&updatedStatus, &updatedApprovalDate)

	if err != nil {
		t.Fatalf("Failed to read updated API request: %v", err)
	}

	if updatedStatus != "approved" {
		t.Errorf("Update failed: expected status 'approved', got '%s'", updatedStatus)
	}

	// The timestamp may not be exactly the same due to database storage, but it should be close
	timeDiff := updatedApprovalDate.Sub(approvalTime)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Update failed: approval date doesn't match expected time (diff: %v)", timeDiff)
	}

	// Test API Request DELETE
	_, err = db.Exec(`DELETE FROM api_requests WHERE id = ?`, requestID)
	if err != nil {
		t.Fatalf("Failed to delete API request: %v", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM api_requests WHERE id = ?`, requestID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check API request deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Delete failed: API request still exists with ID %s", requestID)
	}
}
