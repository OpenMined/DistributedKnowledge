package integration_test

import (
	"dk/db"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

func TestUsageTrackingFinal(t *testing.T) {
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

	// Now create and test a quota notification
	t.Log("Creating test quota notification...")
	notification := &db.QuotaNotification{
		ID:               uuid.New().String(),
		APIID:            testAPIID,
		ExternalUserID:   testUserID,
		NotificationType: "approaching_limit",
		RuleType:         "token",
		PercentageUsed:   80.0,
		Message:          "You're approaching your token limit (80%)",
		CreatedAt:        time.Now(),
		IsRead:           false,
	}

	if err := db.CreateQuotaNotification(database, notification); err != nil {
		t.Fatalf("Failed to create quota notification: %v", err)
	}

	// Retrieve the notification to verify it was saved
	retrievedNotification, err := db.GetQuotaNotification(database, notification.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve quota notification: %v", err)
	}

	t.Logf("Successfully retrieved notification: %s", retrievedNotification.Message)

	// Mark the notification as read
	t.Log("Marking notification as read...")
	if err := db.MarkNotificationAsRead(database, notification.ID); err != nil {
		t.Fatalf("Failed to mark notification as read: %v", err)
	}

	// Verify it was marked as read
	readNotification, err := db.GetQuotaNotification(database, notification.ID)
	if err != nil {
		t.Fatalf("Failed to get notification after marking as read: %v", err)
	}

	if readNotification.IsRead {
		t.Log("Successfully marked notification as read")
	} else {
		t.Error("Failed to mark notification as read")
	}

	t.Log("Usage Tracking Implementation Test Completed Successfully")
}
