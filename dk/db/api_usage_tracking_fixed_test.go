package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// getUniqueTestDB creates a new, empty database file for testing
func getUniqueTestDB(t *testing.T) *sql.DB {
	// Create a unique file name
	uniqueID := uuid.New().String()
	dbFile := "test_db_" + uniqueID + ".db"

	// Open the database
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Add cleanup to remove the file when done
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})

	// Run migrations
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createUniqueTestAPI creates a test API with a unique API key
func createUniqueTestAPI(t *testing.T, db *sql.DB) (string, string) {
	// Generate test IDs
	policyID := uuid.New().String()
	apiID := uuid.New().String()
	uniqueAPIKey := "test_api_key_" + uuid.New().String() // Ensure unique API key

	// Create test policy
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		policyID, "Test Policy", "A policy for testing", "free", true,
		time.Now(), time.Now(), "test_user")
	if err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	// Create test API with unique key
	_, err = db.Exec(`
		INSERT INTO apis (id, name, description, is_active, api_key, host_user_id, policy_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		apiID, "Test API", "An API for testing", true, uniqueAPIKey, "test_host", policyID)
	if err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	return apiID, policyID
}

// TestFixedAPIUsageSummaryOperations tests operations related to API usage summaries
func TestFixedAPIUsageSummaryOperations(t *testing.T) {
	// Skip this test if needed
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Create a unique test database
	db := getUniqueTestDB(t)

	// Create test data with unique API key
	apiID, _ := createUniqueTestAPI(t, db)

	// Test upserting an API usage summary
	t.Run("UpsertAPIUsageSummary", func(t *testing.T) {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

		summary := &APIUsageSummary{
			ID:                uuid.New().String(),
			APIID:             apiID,
			ExternalUserID:    "test_user",
			PeriodType:        "daily",
			PeriodStart:       startOfDay,
			PeriodEnd:         endOfDay,
			TotalRequests:     10,
			TotalTokens:       1000,
			TotalCredits:      1.0,
			TotalTimeMs:       500,
			ThrottledRequests: 1,
			BlockedRequests:   0,
			LastUpdated:       now,
		}

		err := UpsertAPIUsageSummary(db, summary)
		assert.NoError(t, err, "Should upsert API usage summary without error")

		// Test retrieving API usage summaries
		summaries, err := GetAPIUsageSummaries(db, apiID, "test_user", "daily", startOfDay, endOfDay)
		assert.NoError(t, err, "Should retrieve API usage summaries without error")
		assert.Equal(t, 1, len(summaries), "Should have exactly one summary")
		assert.Equal(t, apiID, summaries[0].APIID, "API ID should match")
		assert.Equal(t, "test_user", summaries[0].ExternalUserID, "User ID should match")
		assert.Equal(t, "daily", summaries[0].PeriodType, "Period type should match")
		assert.Equal(t, 10, summaries[0].TotalRequests, "Total requests should match")
		assert.Equal(t, 1000, summaries[0].TotalTokens, "Total tokens should match")
	})

	// Test updating an existing summary
	t.Run("UpdateExistingApiSummary", func(t *testing.T) {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

		// Get the existing summary
		summaries, err := GetAPIUsageSummaries(db, apiID, "test_user", "daily", startOfDay, endOfDay)
		assert.NoError(t, err, "Should retrieve API usage summaries without error")
		assert.Equal(t, 1, len(summaries), "Should have exactly one summary")

		// Update the summary with new values
		summary := summaries[0]
		summary.TotalRequests += 5
		summary.TotalTokens += 500
		summary.TotalCredits += 0.5
		summary.LastUpdated = now

		err = UpsertAPIUsageSummary(db, summary)
		assert.NoError(t, err, "Should update API usage summary without error")

		// Verify the update
		updatedSummaries, err := GetAPIUsageSummaries(db, apiID, "test_user", "daily", startOfDay, endOfDay)
		assert.NoError(t, err, "Should retrieve updated API usage summaries without error")
		assert.Equal(t, 1, len(updatedSummaries), "Should have exactly one summary")
		assert.Equal(t, 15, updatedSummaries[0].TotalRequests, "Total requests should be updated")
		assert.Equal(t, 1500, updatedSummaries[0].TotalTokens, "Total tokens should be updated")
		assert.Equal(t, 1.5, updatedSummaries[0].TotalCredits, "Total credits should be updated")
	})
}

// TestFixedQuotaNotificationOperations tests operations related to quota notifications
func TestFixedQuotaNotificationOperations(t *testing.T) {
	// Skip this test if needed
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Create a unique test database
	db := getUniqueTestDB(t)

	// Create test data with unique API key
	apiID, _ := createUniqueTestAPI(t, db)

	// Test creating a quota notification
	t.Run("CreateQuotaNotification", func(t *testing.T) {
		notification := &QuotaNotification{
			ID:               uuid.New().String(),
			APIID:            apiID,
			ExternalUserID:   "test_user",
			NotificationType: "approaching_limit",
			RuleType:         "token",
			PercentageUsed:   80.0,
			Message:          "You're approaching your token limit (80%)",
			CreatedAt:        time.Now(),
			IsRead:           false,
		}

		err := CreateQuotaNotification(db, notification)
		assert.NoError(t, err, "Should create quota notification without error")

		// Test retrieving a notification by ID
		retrieved, err := GetQuotaNotification(db, notification.ID)
		assert.NoError(t, err, "Should retrieve quota notification without error")
		assert.Equal(t, notification.ID, retrieved.ID, "Notification ID should match")
		assert.Equal(t, apiID, retrieved.APIID, "API ID should match")
		assert.Equal(t, "test_user", retrieved.ExternalUserID, "User ID should match")
		assert.Equal(t, "approaching_limit", retrieved.NotificationType, "Notification type should match")
		assert.Equal(t, "token", retrieved.RuleType, "Rule type should match")
		assert.Equal(t, 80.0, retrieved.PercentageUsed, "Percentage used should match")
		assert.False(t, retrieved.IsRead, "Notification should not be marked as read")
	})

	// Test marking a notification as read
	t.Run("MarkNotificationAsRead", func(t *testing.T) {
		// Get all notifications for the user
		notifications, total, err := GetUserNotifications(db, "test_user", false, 10, 0)
		assert.NoError(t, err, "Should retrieve user notifications without error")
		assert.GreaterOrEqual(t, total, 1, "Should have at least one notification")
		assert.GreaterOrEqual(t, len(notifications), 1, "Should have at least one notification")

		notificationID := notifications[0].ID

		// Mark as read
		err = MarkNotificationAsRead(db, notificationID)
		assert.NoError(t, err, "Should mark notification as read without error")

		// Verify the notification is marked as read
		notification, err := GetQuotaNotification(db, notificationID)
		assert.NoError(t, err, "Should retrieve notification without error")
		assert.True(t, notification.IsRead, "Notification should be marked as read")
		assert.NotNil(t, notification.ReadAt, "Read timestamp should be set")
	})
}

// TestFixedUsageSummaryRefresh tests refreshing usage summaries
func TestFixedUsageSummaryRefresh(t *testing.T) {
	// Skip this test if needed
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Create a unique test database
	db := getUniqueTestDB(t)

	// Create test data with unique API key
	apiID, _ := createUniqueTestAPI(t, db)

	// Insert multiple usage records
	t.Run("InsertMultipleUsageRecords", func(t *testing.T) {
		now := time.Now()
		// Create 5 usage records
		for i := 0; i < 5; i++ {
			usage := &APIUsage{
				ID:              uuid.New().String(),
				APIID:           apiID,
				ExternalUserID:  "test_user",
				Timestamp:       now.Add(time.Duration(-i) * time.Hour), // Spread over last few hours
				RequestCount:    1,
				TokensUsed:      100,
				CreditsConsumed: 0.1,
				ExecutionTimeMs: 50,
				Endpoint:        "/api/test",
				WasThrottled:    false,
				WasBlocked:      false,
			}

			err := RecordAPIUsage(db, usage)
			assert.NoError(t, err, "Should record API usage without error")
		}
	})

	// Test updating summaries
	t.Run("UpdateAPIUsageSummaries", func(t *testing.T) {
		err := UpdateAPIUsageSummaries(db)
		assert.NoError(t, err, "Should update API usage summaries without error")

		// Verify daily summary was created
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

		summaries, err := GetAPIUsageSummaries(db, apiID, "test_user", "daily", startOfDay, endOfDay)
		assert.NoError(t, err, "Should retrieve daily summaries without error")
		assert.GreaterOrEqual(t, len(summaries), 1, "Should have at least one daily summary")
		assert.Equal(t, "daily", summaries[0].PeriodType, "Period type should be daily")
		assert.GreaterOrEqual(t, summaries[0].TotalRequests, 5, "Should have recorded at least 5 requests")
		assert.GreaterOrEqual(t, summaries[0].TotalTokens, 500, "Should have recorded at least 500 tokens")

		// Verify weekly summary was created
		daysSinceMonday := int(now.Weekday())
		if daysSinceMonday == 0 { // Sunday
			daysSinceMonday = 6
		} else {
			daysSinceMonday--
		}
		startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
		endOfWeek := startOfWeek.Add(7 * 24 * time.Hour).Add(-time.Second)

		weeklySummaries, err := GetAPIUsageSummaries(db, apiID, "test_user", "weekly", startOfWeek, endOfWeek)
		assert.NoError(t, err, "Should retrieve weekly summaries without error")
		assert.GreaterOrEqual(t, len(weeklySummaries), 1, "Should have at least one weekly summary")
		assert.Equal(t, "weekly", weeklySummaries[0].PeriodType, "Period type should be weekly")
	})
}
