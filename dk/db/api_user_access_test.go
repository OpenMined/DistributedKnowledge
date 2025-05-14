package db

import (
	"database/sql"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"os"
	"testing"
	"time"
)

// TestAPIUserAccessCRUD tests the CRUD operations for the api_user_access table
func TestAPIUserAccessCRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// First create an API to reference
	apiID := uuid.New().String()
	apiKey := "test_key_" + apiID[0:8] // Generate a unique API key based on the ID
	_, err := db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", true, apiKey, "test_host")

	if err != nil {
		t.Fatalf("Failed to insert API for UserAccess test: %v", err)
	}

	// Test API User Access CREATE
	accessID := uuid.New().String()
	externalUserID := "test_external_user"
	now := time.Now()

	_, err = db.Exec(`
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, accessID, apiID, externalUserID, "read", now, "test_host", true)

	if err != nil {
		t.Fatalf("Failed to insert test API user access: %v", err)
	}

	// Test API User Access READ
	var retrievedAccess struct {
		ID             string
		APIID          string
		ExternalUserID string
		AccessLevel    string
		GrantedBy      string
		IsActive       bool
	}

	err = db.QueryRow(`
		SELECT id, api_id, external_user_id, access_level, granted_by, is_active
		FROM api_user_access
		WHERE id = ?
	`, accessID).Scan(
		&retrievedAccess.ID,
		&retrievedAccess.APIID,
		&retrievedAccess.ExternalUserID,
		&retrievedAccess.AccessLevel,
		&retrievedAccess.GrantedBy,
		&retrievedAccess.IsActive,
	)

	if err != nil {
		t.Fatalf("Failed to read API user access: %v", err)
	}

	// Verify retrieved data
	if retrievedAccess.ID != accessID {
		t.Errorf("Expected access ID %s, got %s", accessID, retrievedAccess.ID)
	}
	if retrievedAccess.APIID != apiID {
		t.Errorf("Expected API ID %s, got %s", apiID, retrievedAccess.APIID)
	}
	if retrievedAccess.ExternalUserID != externalUserID {
		t.Errorf("Expected external user ID %s, got %s", externalUserID, retrievedAccess.ExternalUserID)
	}
	if retrievedAccess.AccessLevel != "read" {
		t.Errorf("Expected access level 'read', got '%s'", retrievedAccess.AccessLevel)
	}
	if retrievedAccess.IsActive != true {
		t.Errorf("Expected is_active true, got %v", retrievedAccess.IsActive)
	}

	// Test API User Access UPDATE - change access level
	_, err = db.Exec(`
		UPDATE api_user_access
		SET access_level = ?
		WHERE id = ?
	`, "write", accessID)

	if err != nil {
		t.Fatalf("Failed to update API user access: %v", err)
	}

	// Verify update
	var updatedAccessLevel string
	err = db.QueryRow(`
		SELECT access_level
		FROM api_user_access
		WHERE id = ?
	`, accessID).Scan(&updatedAccessLevel)

	if err != nil {
		t.Fatalf("Failed to read updated API user access: %v", err)
	}

	if updatedAccessLevel != "write" {
		t.Errorf("Update failed: expected access_level 'write', got '%s'", updatedAccessLevel)
	}

	// Test Revoke Access (not DELETE but soft delete by setting is_active to false)
	revokeTime := time.Now()
	_, err = db.Exec(`
		UPDATE api_user_access
		SET is_active = ?, revoked_at = ?
		WHERE id = ?
	`, false, revokeTime, accessID)

	if err != nil {
		t.Fatalf("Failed to revoke API user access: %v", err)
	}

	// Verify revocation
	var isActive bool
	var revokedAt time.Time
	err = db.QueryRow(`
		SELECT is_active, revoked_at
		FROM api_user_access
		WHERE id = ?
	`, accessID).Scan(&isActive, &revokedAt)

	if err != nil {
		t.Fatalf("Failed to read revoked API user access: %v", err)
	}

	if isActive != false {
		t.Errorf("Revoke failed: expected is_active false, got %v", isActive)
	}

	// The timestamp may not be exactly the same due to database storage, but it should be close
	timeDiff := revokedAt.Sub(revokeTime)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Revoke failed: revoked_at doesn't match expected time (diff: %v)", timeDiff)
	}

	// Test Restore Access
	_, err = db.Exec(`
		UPDATE api_user_access
		SET is_active = ?, revoked_at = NULL
		WHERE id = ?
	`, true, accessID)

	if err != nil {
		t.Fatalf("Failed to restore API user access: %v", err)
	}

	// Verify restoration
	var nullableRevokedAt sql.NullTime
	err = db.QueryRow(`
		SELECT is_active, revoked_at
		FROM api_user_access
		WHERE id = ?
	`, accessID).Scan(&isActive, &nullableRevokedAt)

	if err != nil {
		t.Fatalf("Failed to read restored API user access: %v", err)
	}

	if isActive != true {
		t.Errorf("Restore failed: expected is_active true, got %v", isActive)
	}

	if nullableRevokedAt.Valid {
		t.Errorf("Restore failed: expected revoked_at to be NULL, but it has value %v", nullableRevokedAt.Time)
	}

	// Test hard DELETE (normally we would just do a soft delete)
	_, err = db.Exec(`DELETE FROM api_user_access WHERE id = ?`, accessID)
	if err != nil {
		t.Fatalf("Failed to delete API user access: %v", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM api_user_access WHERE id = ?`, accessID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check API user access deletion: %v", err)
	}
	if count != 0 {
		t.Errorf("Delete failed: API user access still exists with ID %s", accessID)
	}
}

// TestListAPIUserAccess tests the functionality to list API user access records
func TestListAPIUserAccess(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Create an API for testing
	apiID := uuid.New().String()
	apiKey := "test_key_" + apiID[0:8] // Generate a unique API key based on the ID
	_, err := db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", true, apiKey, "test_host")

	if err != nil {
		t.Fatalf("Failed to insert API for list test: %v", err)
	}

	// Insert multiple user access records
	users := []struct {
		id          string
		userID      string
		accessLevel string
		isActive    bool
		grantedAt   time.Time
	}{
		{uuid.New().String(), "user1", "read", true, time.Now().Add(-10 * time.Hour)},
		{uuid.New().String(), "user2", "write", true, time.Now().Add(-5 * time.Hour)},
		{uuid.New().String(), "user3", "admin", true, time.Now()},
		{uuid.New().String(), "user4", "read", false, time.Now().Add(-15 * time.Hour)}, // Inactive user
	}

	// Insert the test data
	for _, user := range users {
		var revokedAt interface{}
		if !user.isActive {
			revokedTime := time.Now().Add(-1 * time.Hour)
			revokedAt = revokedTime
		}

		_, err := db.Exec(`
			INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, user.id, apiID, user.userID, user.accessLevel, user.grantedAt, "test_host", revokedAt, user.isActive)

		if err != nil {
			t.Fatalf("Failed to insert test user access: %v", err)
		}
	}

	// Test listing only active users
	accessRecords, total, err := ListAPIUserAccess(db, apiID, true, 10, 0, "granted_at", "desc")
	if err != nil {
		t.Fatalf("Failed to list API user access: %v", err)
	}

	// Should have 3 active records
	if total != 3 {
		t.Errorf("Expected 3 active records, got %d", total)
	}
	if len(accessRecords) != 3 {
		t.Errorf("Expected 3 active records returned, got %d", len(accessRecords))
	}

	// Verify sorting (descending by granted_at means user3, user2, user1)
	if len(accessRecords) >= 3 {
		if accessRecords[0].ExternalUserID != "user3" {
			t.Errorf("Expected first user to be user3, got %s", accessRecords[0].ExternalUserID)
		}
		if accessRecords[2].ExternalUserID != "user1" {
			t.Errorf("Expected third user to be user1, got %s", accessRecords[2].ExternalUserID)
		}
	}

	// Test listing all users (including inactive)
	accessRecords, total, err = ListAPIUserAccess(db, apiID, false, 10, 0, "granted_at", "desc")
	if err != nil {
		t.Fatalf("Failed to list all API user access: %v", err)
	}

	// Should have 4 total records
	if total != 4 {
		t.Errorf("Expected 4 total records, got %d", total)
	}
	if len(accessRecords) != 4 {
		t.Errorf("Expected 4 total records returned, got %d", len(accessRecords))
	}

	// Test pagination
	accessRecords, total, err = ListAPIUserAccess(db, apiID, true, 2, 0, "granted_at", "desc")
	if err != nil {
		t.Fatalf("Failed to list paginated API user access: %v", err)
	}

	// Should have 3 total records but only 2 returned due to limit
	if total != 3 {
		t.Errorf("Expected 3 total active records, got %d", total)
	}
	if len(accessRecords) != 2 {
		t.Errorf("Expected 2 records returned (due to limit), got %d", len(accessRecords))
	}

	// Verify sort order (asc)
	accessRecords, _, err = ListAPIUserAccess(db, apiID, true, 10, 0, "granted_at", "asc")
	if err != nil {
		t.Fatalf("Failed to list ascending API user access: %v", err)
	}

	if len(accessRecords) >= 3 {
		if accessRecords[0].ExternalUserID != "user1" {
			t.Errorf("Expected first user to be user1 with asc sort, got %s", accessRecords[0].ExternalUserID)
		}
		if accessRecords[2].ExternalUserID != "user3" {
			t.Errorf("Expected third user to be user3 with asc sort, got %s", accessRecords[2].ExternalUserID)
		}
	}
}

// TestGetAPIUserAccessByUserID tests retrieving a user access record by API ID and user ID
func TestGetAPIUserAccessByUserID(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Create an API for testing
	apiID := uuid.New().String()
	apiKey := "test_key_" + apiID[0:8] // Generate a unique API key based on the ID
	_, err := db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", true, apiKey, "test_host")

	if err != nil {
		t.Fatalf("Failed to insert API for test: %v", err)
	}

	// Create a user access record
	accessID := uuid.New().String()
	testUserID := "test_user_lookup"
	now := time.Now()

	_, err = db.Exec(`
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, accessID, apiID, testUserID, "read", now, "test_host", true)

	if err != nil {
		t.Fatalf("Failed to insert test API user access: %v", err)
	}

	// Test lookup by API ID and user ID
	access, err := GetAPIUserAccessByUserID(db, apiID, testUserID)
	if err != nil {
		t.Fatalf("Failed to get API user access by user ID: %v", err)
	}

	// Verify retrieved data
	if access.ID != accessID {
		t.Errorf("Expected access ID %s, got %s", accessID, access.ID)
	}
	if access.APIID != apiID {
		t.Errorf("Expected API ID %s, got %s", apiID, access.APIID)
	}
	if access.ExternalUserID != testUserID {
		t.Errorf("Expected external user ID %s, got %s", testUserID, access.ExternalUserID)
	}

	// Test lookup with non-existent user ID
	_, err = GetAPIUserAccessByUserID(db, apiID, "non_existent_user")
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Test lookup with non-existent API ID
	_, err = GetAPIUserAccessByUserID(db, "non_existent_api", testUserID)
	if err == nil {
		t.Error("Expected error for non-existent API, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestUpdateAPIUserAccess tests updating user access permissions
func TestUpdateAPIUserAccess(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Create an API for testing
	apiID := uuid.New().String()
	apiKey := "test_key_" + apiID[0:8] // Generate a unique API key based on the ID
	_, err := db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", true, apiKey, "test_host")

	if err != nil {
		t.Fatalf("Failed to insert API for test: %v", err)
	}

	// Create a user access record
	accessID := uuid.New().String()
	testUserID := "test_user_update"

	access := &APIUserAccess{
		ID:             accessID,
		APIID:          apiID,
		ExternalUserID: testUserID,
		AccessLevel:    "read",
		GrantedAt:      time.Now(),
		GrantedBy:      "test_host",
		IsActive:       true,
	}

	if err := CreateAPIUserAccess(db, access); err != nil {
		t.Fatalf("Failed to create API user access: %v", err)
	}

	// Update the access level
	access.AccessLevel = "write"

	if err := UpdateAPIUserAccess(db, access); err != nil {
		t.Fatalf("Failed to update API user access: %v", err)
	}

	// Retrieve and verify update
	updatedAccess, err := GetAPIUserAccess(db, accessID)
	if err != nil {
		t.Fatalf("Failed to get updated API user access: %v", err)
	}

	if updatedAccess.AccessLevel != "write" {
		t.Errorf("Update failed: expected access_level 'write', got '%s'", updatedAccess.AccessLevel)
	}

	// Test revoking access
	access.IsActive = false
	revokeTime := time.Now()
	access.RevokedAt = &revokeTime

	if err := UpdateAPIUserAccess(db, access); err != nil {
		t.Fatalf("Failed to revoke API user access: %v", err)
	}

	// Retrieve and verify revocation
	revokedAccess, err := GetAPIUserAccess(db, accessID)
	if err != nil {
		t.Fatalf("Failed to get revoked API user access: %v", err)
	}

	if revokedAccess.IsActive {
		t.Error("Revocation failed: expected is_active false, got true")
	}
	if revokedAccess.RevokedAt == nil {
		t.Error("Revocation failed: revoked_at is nil")
	}

	// Test restoring access
	access.IsActive = true
	access.RevokedAt = nil

	if err := UpdateAPIUserAccess(db, access); err != nil {
		t.Fatalf("Failed to restore API user access: %v", err)
	}

	// Retrieve and verify restoration
	restoredAccess, err := GetAPIUserAccess(db, accessID)
	if err != nil {
		t.Fatalf("Failed to get restored API user access: %v", err)
	}

	if !restoredAccess.IsActive {
		t.Error("Restoration failed: expected is_active true, got false")
	}
	if restoredAccess.RevokedAt != nil {
		t.Error("Restoration failed: revoked_at is not nil")
	}

	// Test updating non-existent record
	nonExistentAccess := &APIUserAccess{
		ID:             uuid.New().String(),
		APIID:          apiID,
		ExternalUserID: "non_existent_user",
		AccessLevel:    "read",
	}

	err = UpdateAPIUserAccess(db, nonExistentAccess)
	if err == nil {
		t.Error("Expected error for non-existent access record, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestAPIAccessLevels tests the enforcement of valid access levels
func TestAPIAccessLevels(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Create an API for testing
	apiID := uuid.New().String()
	apiKey := "test_key_" + apiID[0:8] // Generate a unique API key based on the ID
	_, err := db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, apiID, "Test API", "API for testing", true, apiKey, "test_host")

	if err != nil {
		t.Fatalf("Failed to insert API for test: %v", err)
	}

	// Test cases for access levels
	testCases := []struct {
		accessLevel string
		userID      string
		shouldFail  bool
	}{
		{"read", "user_read", false},      // Valid
		{"write", "user_write", false},    // Valid
		{"admin", "user_admin", false},    // Valid
		{"invalid", "user_invalid", true}, // Invalid - doesn't match CHECK constraint
		{"", "user_empty", true},          // Invalid - empty
	}

	for _, tc := range testCases {
		t.Run("AccessLevel_"+tc.accessLevel, func(t *testing.T) {
			accessID := uuid.New().String()
			_, err := db.Exec(`
				INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, is_active)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, accessID, apiID, tc.userID, tc.accessLevel, time.Now(), "test_host", true)

			if tc.shouldFail && err == nil {
				t.Errorf("Expected insertion with access level '%s' to fail, but it succeeded", tc.accessLevel)
			} else if !tc.shouldFail && err != nil {
				t.Errorf("Expected insertion with access level '%s' to succeed, but it failed: %v", tc.accessLevel, err)
			}
		})
	}
}
