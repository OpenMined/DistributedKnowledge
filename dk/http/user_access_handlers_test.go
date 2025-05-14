package http

import (
	"bytes"
	"context"
	"database/sql"
	"dk/db"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// setupTestDBAndContext creates a test database and returns a context with the database
func setupTestDBAndContext(t *testing.T) (context.Context, *sql.DB) {
	// Create a test database
	testDB, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Set up necessary tables
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create the APIs table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS apis (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT FALSE,
			api_key TEXT UNIQUE,
			host_user_id TEXT NOT NULL,
			policy_id TEXT,
			is_deprecated BOOLEAN DEFAULT FALSE,
			deprecation_date DATETIME,
			deprecation_message TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create apis table: %v", err)
	}

	// Create API user access table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS api_user_access (
			id TEXT PRIMARY KEY,
			api_id TEXT NOT NULL,
			external_user_id TEXT NOT NULL,
			access_level TEXT NOT NULL DEFAULT 'read' CHECK (access_level IN ('read', 'write', 'admin')),
			granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			granted_by TEXT,
			revoked_at DATETIME,
			is_active BOOLEAN DEFAULT TRUE,
			FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE,
			UNIQUE (api_id, external_user_id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create api_user_access table: %v", err)
	}

	// Create a context with the database
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", testDB) // Using string key instead of utils.DBContextKey

	// Add a mock user ID to the context (simulating an authenticated request)
	ctx = context.WithValue(ctx, "user_id", "local-user") // Using string key instead of utils.UserIDContextKey

	return ctx, testDB
}

// setupTestAPI creates a test API in the database
func setupTestAPI(t *testing.T, testDB *sql.DB) *db.API {
	apiID := uuid.New().String()
	api := &db.API{
		ID:          apiID,
		Name:        "Test API",
		Description: "API for testing",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		APIKey:      "test_api_key_" + apiID[0:8], // Generate a unique API key
		HostUserID:  "local-user",
	}

	err := db.CreateAPI(testDB, api)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	return api
}

// setupTestAPIUserAccess creates a test API user access record
func setupTestAPIUserAccess(t *testing.T, testDB *sql.DB, apiID, userID, accessLevel string, isActive bool) *db.APIUserAccess {
	access := &db.APIUserAccess{
		ID:             uuid.New().String(),
		APIID:          apiID,
		ExternalUserID: userID,
		AccessLevel:    accessLevel,
		GrantedAt:      time.Now(),
		GrantedBy:      "local-user",
		IsActive:       isActive,
	}

	if !isActive {
		revokedAt := time.Now()
		access.RevokedAt = &revokedAt
	}

	err := db.CreateAPIUserAccess(testDB, access)
	if err != nil {
		t.Fatalf("Failed to create test API user access: %v", err)
	}

	return access
}

// TestHandleGetAPIUsers tests the GetAPIUsers handler
func TestHandleGetAPIUsers(t *testing.T) {
	ctx, testDB := setupTestDBAndContext(t)

	// Create a test API
	api := setupTestAPI(t, testDB)

	// Create some test user access records
	_ = setupTestAPIUserAccess(t, testDB, api.ID, "user1", "read", true)
	_ = setupTestAPIUserAccess(t, testDB, api.ID, "user2", "write", true)
	_ = setupTestAPIUserAccess(t, testDB, api.ID, "user3", "admin", false) // inactive user

	// Create a request
	req, err := http.NewRequest("GET", "/api/apis/"+api.ID+"/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HandleGetAPIUsers(ctx, rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var response APIUserListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// By default, should only return active users
	if len(response.Users) != 2 {
		t.Errorf("Expected 2 active users, got %d", len(response.Users))
	}

	// Test with active=false query parameter
	req, _ = http.NewRequest("GET", "/api/apis/"+api.ID+"/users?active=false", nil)
	rr = httptest.NewRecorder()
	HandleGetAPIUsers(ctx, rr, req)

	// Parse the response
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return all users, including inactive
	if len(response.Users) != 3 {
		t.Errorf("Expected 3 total users, got %d", len(response.Users))
	}

	// Test with invalid API ID
	req, _ = http.NewRequest("GET", "/api/apis/invalid-id/users", nil)
	rr = httptest.NewRecorder()
	HandleGetAPIUsers(ctx, rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for invalid API ID: got %v want %v", status, http.StatusNotFound)
	}
}

// TestHandleGrantAPIAccess tests the GrantAPIAccess handler
func TestHandleGrantAPIAccess(t *testing.T) {
	ctx, testDB := setupTestDBAndContext(t)

	// Create a test API
	api := setupTestAPI(t, testDB)

	// Create request body
	reqBody := APIUserAccessRequest{
		UserID:      "new-test-user",
		AccessLevel: "read",
	}

	reqBodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/apis/"+api.ID+"/users", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HandleGrantAPIAccess(ctx, rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Parse the response
	var response APIUserAccessResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the response
	if response.UserID != "new-test-user" {
		t.Errorf("Expected user ID new-test-user, got %s", response.UserID)
	}
	if response.AccessLevel != "read" {
		t.Errorf("Expected access level read, got %s", response.AccessLevel)
	}
	if !response.IsActive {
		t.Errorf("Expected is_active true, got %v", response.IsActive)
	}

	// Test granting access to an existing user
	req, _ = http.NewRequest("POST", "/api/apis/"+api.ID+"/users", bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleGrantAPIAccess(ctx, rr, req)

	// Should return 409 Conflict
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("Handler returned wrong status code for existing user: got %v want %v", status, http.StatusConflict)
	}

	// Test reactivating a revoked user
	inactiveUser := setupTestAPIUserAccess(t, testDB, api.ID, "inactive-user", "read", false)
	if inactiveUser == nil {
		t.Fatalf("Failed to create inactive test user")
	}

	reqBody = APIUserAccessRequest{
		UserID:      "inactive-user",
		AccessLevel: "write", // Changing the access level
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/apis/"+api.ID+"/users", bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleGrantAPIAccess(ctx, rr, req)

	// Should return 200 OK (for update)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code for reactivation: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the reactivation
	if response.UserID != "inactive-user" {
		t.Errorf("Expected user ID inactive-user, got %s", response.UserID)
	}
	if response.AccessLevel != "write" {
		t.Errorf("Expected access level write, got %s", response.AccessLevel)
	}
	if !response.IsActive {
		t.Errorf("Expected is_active true, got %v", response.IsActive)
	}
	if response.RevokedAt != nil {
		t.Errorf("Expected revoked_at nil, got %v", response.RevokedAt)
	}

	// Test with invalid access level
	reqBody = APIUserAccessRequest{
		UserID:      "another-user",
		AccessLevel: "invalid",
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/apis/"+api.ID+"/users", bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleGrantAPIAccess(ctx, rr, req)

	// Should return 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code for invalid access level: got %v want %v", status, http.StatusBadRequest)
	}

	// Test with invalid API ID
	reqBody = APIUserAccessRequest{
		UserID:      "new-user",
		AccessLevel: "read",
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/apis/invalid-id/users", bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleGrantAPIAccess(ctx, rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for invalid API ID: got %v want %v", status, http.StatusNotFound)
	}
}

// TestHandleUpdateAPIUserAccess tests the UpdateAPIUserAccess handler
func TestHandleUpdateAPIUserAccess(t *testing.T) {
	ctx, testDB := setupTestDBAndContext(t)

	// Create a test API
	api := setupTestAPI(t, testDB)

	// Create a test user access record
	access := setupTestAPIUserAccess(t, testDB, api.ID, "update-test-user", "read", true)

	// Create request body to update access level
	reqBody := APIUserAccessUpdateRequest{
		AccessLevel: "admin",
	}

	reqBodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("PATCH", "/api/apis/"+api.ID+"/users/"+access.ExternalUserID, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HandleUpdateAPIUserAccess(ctx, rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var response APIUserAccessResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the updated access level
	if response.AccessLevel != "admin" {
		t.Errorf("Expected access level admin, got %s", response.AccessLevel)
	}

	// Test updating an inactive user
	inactiveUser := setupTestAPIUserAccess(t, testDB, api.ID, "inactive-update-user", "read", false)

	reqBody = APIUserAccessUpdateRequest{
		AccessLevel: "write",
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("PATCH", "/api/apis/"+api.ID+"/users/"+inactiveUser.ExternalUserID, bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleUpdateAPIUserAccess(ctx, rr, req)

	// Should return 400 Bad Request (can't update inactive user)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code for inactive user: got %v want %v", status, http.StatusBadRequest)
	}

	// Test with invalid access level
	reqBody = APIUserAccessUpdateRequest{
		AccessLevel: "invalid",
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("PATCH", "/api/apis/"+api.ID+"/users/"+access.ExternalUserID, bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleUpdateAPIUserAccess(ctx, rr, req)

	// Should return 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code for invalid access level: got %v want %v", status, http.StatusBadRequest)
	}

	// Test with nonexistent user
	reqBody = APIUserAccessUpdateRequest{
		AccessLevel: "read",
	}

	reqBodyBytes, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("PATCH", "/api/apis/"+api.ID+"/users/nonexistent-user", bytes.NewBuffer(reqBodyBytes))
	rr = httptest.NewRecorder()
	HandleUpdateAPIUserAccess(ctx, rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for nonexistent user: got %v want %v", status, http.StatusNotFound)
	}
}

// TestHandleRevokeAPIUserAccess tests the RevokeAPIUserAccess handler
func TestHandleRevokeAPIUserAccess(t *testing.T) {
	ctx, testDB := setupTestDBAndContext(t)

	// Create a test API
	api := setupTestAPI(t, testDB)

	// Create a test user access record
	access := setupTestAPIUserAccess(t, testDB, api.ID, "revoke-test-user", "read", true)

	// Create DELETE request
	req, err := http.NewRequest("DELETE", "/api/apis/"+api.ID+"/users/"+access.ExternalUserID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HandleRevokeAPIUserAccess(ctx, rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var response APIUserAccessResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the revocation
	if response.IsActive {
		t.Errorf("Expected is_active false, got %v", response.IsActive)
	}
	if response.RevokedAt == nil {
		t.Errorf("Expected revoked_at to be set, got nil")
	}

	// Try to revoke the same user again
	req, _ = http.NewRequest("DELETE", "/api/apis/"+api.ID+"/users/"+access.ExternalUserID, nil)
	rr = httptest.NewRecorder()
	HandleRevokeAPIUserAccess(ctx, rr, req)

	// Should return 400 Bad Request (already revoked)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code for already revoked user: got %v want %v", status, http.StatusBadRequest)
	}

	// Test with nonexistent user
	req, _ = http.NewRequest("DELETE", "/api/apis/"+api.ID+"/users/nonexistent-user", nil)
	rr = httptest.NewRecorder()
	HandleRevokeAPIUserAccess(ctx, rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for nonexistent user: got %v want %v", status, http.StatusNotFound)
	}
}

// TestHandleRestoreAPIUserAccess tests the RestoreAPIUserAccess handler
func TestHandleRestoreAPIUserAccess(t *testing.T) {
	ctx, testDB := setupTestDBAndContext(t)

	// Create a test API
	api := setupTestAPI(t, testDB)

	// Create an inactive user access record
	inactiveUser := setupTestAPIUserAccess(t, testDB, api.ID, "restore-test-user", "read", false)

	// Create POST request for restoration
	req, err := http.NewRequest("POST", "/api/apis/"+api.ID+"/users/"+inactiveUser.ExternalUserID+"/restore", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	HandleRestoreAPIUserAccess(ctx, rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response
	var response APIUserAccessResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the restoration
	if !response.IsActive {
		t.Errorf("Expected is_active true, got %v", response.IsActive)
	}
	if response.RevokedAt != nil {
		t.Errorf("Expected revoked_at nil, got %v", response.RevokedAt)
	}

	// Create an active user access record
	activeUser := setupTestAPIUserAccess(t, testDB, api.ID, "active-test-user", "read", true)

	// Try to restore an already active user
	req, _ = http.NewRequest("POST", "/api/apis/"+api.ID+"/users/"+activeUser.ExternalUserID+"/restore", nil)
	rr = httptest.NewRecorder()
	HandleRestoreAPIUserAccess(ctx, rr, req)

	// Should return 400 Bad Request (already active)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code for already active user: got %v want %v", status, http.StatusBadRequest)
	}

	// Test with nonexistent user
	req, _ = http.NewRequest("POST", "/api/apis/"+api.ID+"/users/nonexistent-user/restore", nil)
	rr = httptest.NewRecorder()
	HandleRestoreAPIUserAccess(ctx, rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code for nonexistent user: got %v want %v", status, http.StatusNotFound)
	}
}
