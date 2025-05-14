package integration_test

import (
	"dk/db"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

func TestUsageTracking(t *testing.T) {
	// Remove any existing test database
	os.Remove("test_usage_tracking.db")

	// Initialize the database connection
	database, err := db.Initialize("test_usage_tracking.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	defer os.Remove("test_usage_tracking.db") // Clean up test database when done

	// Run migrations to create tables
	if err := db.RunAPIMigrations(database); err != nil {
		t.Fatalf("Failed to run API Management migrations: %v", err)
	}

	t.Log("Successfully initialized database and ran migrations")

	// Create test data for API usage
	testAPIID := uuid.New().String()
	testUserID := "test_user"

	// Create a test API with a unique API key
	api := &db.API{
		ID:          testAPIID,
		Name:        "Test API " + uuid.New().String()[0:8],
		Description: "API for testing usage tracking",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		APIKey:      "test_key_" + uuid.New().String(),
		HostUserID:  "host_user",
	}

	t.Log("Creating test API...")
	if err := db.CreateAPI(database, api); err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Directly test recording a single usage record
	t.Log("Recording single API usage...")
	usage := &db.APIUsage{
		ID:              uuid.New().String(),
		APIID:           testAPIID,
		ExternalUserID:  testUserID,
		Timestamp:       time.Now(),
		RequestCount:    1,
		TokensUsed:      100,
		CreditsConsumed: 0.1,
		ExecutionTimeMs: 50,
		Endpoint:        "/api/test",
		WasThrottled:    false,
		WasBlocked:      false,
	}

	if err := db.RecordAPIUsage(database, usage); err != nil {
		t.Fatalf("Failed to record API usage: %v", err)
	}

	// Verify the record was created using GetRecentAPIUsage
	recentUsages, err := db.GetRecentAPIUsage(database, testAPIID, testUserID, 10)
	if err != nil {
		t.Fatalf("Failed to retrieve recent API usage: %v", err)
	}

	if len(recentUsages) == 0 || recentUsages[0].ID != usage.ID {
		t.Errorf("Retrieved usage doesn't match original")
	} else {
		t.Log("Successfully retrieved API usage record")
	}

	// Test getting usage by period
	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now().Add(24 * time.Hour)

	usages, err := db.GetUsageByPeriod(database, testAPIID, testUserID, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to get usage by period: %v", err)
	}

	if len(usages) == 0 {
		t.Errorf("No usage records found for API ID %s", testAPIID)
	} else {
		t.Logf("Found %d usage records for API ID %s", len(usages), testAPIID)
	}

	// Test getting total usage for a period
	usageSummary, err := db.GetTotalUsageForPeriod(database, testAPIID, testUserID, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to get total usage: %v", err)
	}

	if usageSummary.TotalRequests == 0 {
		t.Errorf("No usage summary found for user ID %s", testUserID)
	} else {
		t.Logf("Total requests for period: %d", usageSummary.TotalRequests)
	}

	t.Log("Basic Usage Tracking Test Completed Successfully")
}
