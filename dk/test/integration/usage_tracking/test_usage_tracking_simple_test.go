package integration_test

import (
	"dk/db"
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

func TestUsageTrackingSimple(t *testing.T) {
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

	// Create test policy
	policyID := uuid.New().String()
	policy := &db.Policy{
		ID:          policyID,
		Name:        "Test Rate Limit Policy",
		Description: "Policy for testing rate limits",
		Type:        "rate",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   "test_admin",
	}

	t.Log("Creating test policy...")
	if err := db.CreatePolicy(database, policy); err != nil {
		t.Fatalf("Failed to create test policy: %v", err)
	}

	// Create a policy rule
	ruleID := uuid.New().String()
	rule := &db.PolicyRule{
		ID:         ruleID,
		PolicyID:   policyID,
		RuleType:   "rate",
		LimitValue: 100.0,
		Period:     "day",
		Action:     "block",
		Priority:   10,
		CreatedAt:  time.Now(),
	}

	t.Log("Creating policy rule...")
	if err := db.CreatePolicyRule(database, rule); err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Create a test API with the policy
	apiID := uuid.New().String()
	api := &db.API{
		ID:          apiID,
		Name:        "Rate Limited API",
		Description: "API for testing rate limiting",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		APIKey:      "test_key_" + uuid.New().String(),
		HostUserID:  "host_user",
		PolicyID:    &policyID,
	}

	t.Log("Creating test API...")
	if err := db.CreateAPI(database, api); err != nil {
		t.Fatalf("Failed to create test API: %v", err)
	}

	// Test recording 10 usage records for simulating rate limit
	t.Log("Recording multiple API usage records...")
	testUserID := "test_user"
	for i := 0; i < 10; i++ {
		usage := &db.APIUsage{
			ID:              uuid.New().String(),
			APIID:           apiID,
			ExternalUserID:  testUserID,
			Timestamp:       time.Now(),
			RequestCount:    1,
			TokensUsed:      10,
			CreditsConsumed: 0.01,
			ExecutionTimeMs: 20,
			Endpoint:        "/api/test",
			WasThrottled:    false,
			WasBlocked:      false,
		}

		if err := db.RecordAPIUsage(database, usage); err != nil {
			t.Fatalf("Failed to record API usage: %v", err)
		}
	}

	// Check total usage for a period
	periodStart := time.Now().Add(-24 * time.Hour)
	periodEnd := time.Now().Add(24 * time.Hour)

	usageSummary, err := db.GetTotalUsageForPeriod(database, apiID, testUserID, periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Failed to get total usage: %v", err)
	}

	t.Logf("Total requests for period: %d", usageSummary.TotalRequests)
	if usageSummary.TotalRequests != 10 {
		t.Errorf("Expected 10 requests, got %d", usageSummary.TotalRequests)
	}

	// Test quota notification
	notificationID := uuid.New().String()
	notification := &db.QuotaNotification{
		ID:               notificationID,
		APIID:            apiID,
		ExternalUserID:   testUserID,
		NotificationType: "approaching_limit",
		RuleType:         "rate",
		PercentageUsed:   10.0, // 10 out of 100 requests
		Message:          "You've used 10% of your daily request limit",
		CreatedAt:        time.Now(),
		IsRead:           false,
	}

	t.Log("Creating quota notification...")
	if err := db.CreateQuotaNotification(database, notification); err != nil {
		t.Fatalf("Failed to create quota notification: %v", err)
	}

	// Get user notifications
	notifications, total, err := db.GetUserNotifications(database, testUserID, true, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get user notifications: %v", err)
	}

	if len(notifications) == 0 {
		t.Errorf("No unread notifications found")
	} else {
		t.Logf("Found %d unread notifications out of %d total", len(notifications), total)
	}

	// Mark notification as read
	if err := db.MarkNotificationAsRead(database, notificationID); err != nil {
		t.Fatalf("Failed to mark notification as read: %v", err)
	}

	// Verify notification is marked as read
	updatedNotification, err := db.GetQuotaNotification(database, notificationID)
	if err != nil {
		t.Fatalf("Failed to get updated notification: %v", err)
	}

	if !updatedNotification.IsRead {
		t.Errorf("Notification was not marked as read")
	} else {
		t.Log("Notification successfully marked as read")
	}

	// Test usage summary
	summaryID := uuid.New().String()
	summary := &db.APIUsageSummary{
		ID:                summaryID,
		APIID:             apiID,
		ExternalUserID:    testUserID,
		PeriodType:        "daily",
		PeriodStart:       time.Now().Truncate(24 * time.Hour),
		PeriodEnd:         time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
		TotalRequests:     10,
		TotalTokens:       100,
		TotalCredits:      0.1,
		TotalTimeMs:       200,
		ThrottledRequests: 0,
		BlockedRequests:   0,
		LastUpdated:       time.Now(),
	}

	t.Log("Creating usage summary...")
	if err := db.UpsertAPIUsageSummary(database, summary); err != nil {
		t.Fatalf("Failed to create usage summary: %v", err)
	}

	// Get usage summary for period
	periodStartSummary := time.Now().Add(-48 * time.Hour)
	periodEndSummary := time.Now().Add(48 * time.Hour)
	summaries, err := db.GetAPIUsageSummaries(database, apiID, testUserID, "daily", periodStartSummary, periodEndSummary)
	if err != nil {
		t.Fatalf("Failed to get usage summaries: %v", err)
	}

	if len(summaries) == 0 {
		t.Errorf("No usage summaries found")
	} else {
		t.Logf("Found %d usage summaries", len(summaries))
	}

	t.Log("Simple Usage Tracking Test Completed Successfully")
}
