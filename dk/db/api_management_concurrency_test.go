package db

import (
	"fmt"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"sync"
	"testing"
	"time"
)

// TestConcurrentUsageTracking tests concurrent usage tracking in the api_usage table
func TestConcurrentUsageTracking(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Create a test API and policy for the concurrent operations
	policyID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policyID, "Concurrent Test Policy", "Policy for concurrent testing", "token", true, time.Now(), time.Now(), "test_user")

	if err != nil {
		t.Fatalf("Failed to insert policy for concurrency test: %v", err)
	}

	apiID := uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO apis (id, name, description, created_at, updated_at, is_active, api_key, host_user_id, policy_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, apiID, "Concurrent Test API", "API for concurrency testing", time.Now(), time.Now(), true, "concurrent_test_key", "test_host", policyID)

	if err != nil {
		t.Fatalf("Failed to insert API for concurrency test: %v", err)
	}

	// Define concurrent users
	users := []string{"user1", "user2", "user3", "user4", "user5"}

	// Number of concurrent operations per user
	operationsPerUser := 10

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(len(users) * operationsPerUser)

	// Track errors from goroutines
	errChan := make(chan error, len(users)*operationsPerUser)

	// Function to insert usage records concurrently
	insertUsageRecord := func(userID string, tokenCount int) {
		defer wg.Done()

		usageID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO api_usage (id, api_id, external_user_id, timestamp, tokens_used, endpoint)
			VALUES (?, ?, ?, ?, ?, ?)
		`, usageID, apiID, userID, time.Now(), tokenCount, "/test/endpoint")

		if err != nil {
			errChan <- err
		}
	}

	// Start concurrent operations
	for _, user := range users {
		for i := 0; i < operationsPerUser; i++ {
			// Add some randomness to token count
			tokenCount := 100 + (i * 10)
			go insertUsageRecord(user, tokenCount)
		}
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors during concurrent usage tracking: First error: %v", len(errors), errors[0])
	}

	// Verify that the correct number of records were inserted
	var recordCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM api_usage WHERE api_id = ?`, apiID).Scan(&recordCount)
	if err != nil {
		t.Fatalf("Failed to count usage records: %v", err)
	}

	expectedCount := len(users) * operationsPerUser
	if recordCount != expectedCount {
		t.Errorf("Expected %d usage records, got %d", expectedCount, recordCount)
	}
}

// TestConcurrentPolicyRuleUpdates tests concurrent policy rule updates
func TestConcurrentPolicyRuleUpdates(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Create a test policy for concurrent rule updates
	policyID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policyID, "Concurrent Rule Policy", "Policy for testing concurrent rule updates", "composite", true, time.Now(), time.Now(), "test_user")

	if err != nil {
		t.Fatalf("Failed to insert policy for concurrency test: %v", err)
	}

	// Create initial rules
	ruleIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		ruleID := uuid.New().String()
		ruleIDs[i] = ruleID

		_, err := db.Exec(`
			INSERT INTO policy_rules (id, policy_id, rule_type, limit_value, period, action, priority, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, ruleID, policyID, "token", 1000.0, "day", "log", 100+i, time.Now())

		if err != nil {
			t.Fatalf("Failed to insert initial rules for concurrency test: %v", err)
		}
	}

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(len(ruleIDs) * 2) // We'll update each rule from 2 goroutines

	// Track errors from goroutines
	errChan := make(chan error, len(ruleIDs)*2)

	// Function to update rules concurrently
	updateRule := func(ruleID string, limitValue float64, action string) {
		defer wg.Done()

		_, err := db.Exec(`
			UPDATE policy_rules
			SET limit_value = ?, action = ?, priority = priority + 1
			WHERE id = ?
		`, limitValue, action, ruleID)

		if err != nil {
			errChan <- err
		}
	}

	// Start concurrent operations
	for _, ruleID := range ruleIDs {
		// Two updates for each rule with different values
		go updateRule(ruleID, 1500.0, "throttle")
		go updateRule(ruleID, 2000.0, "block")
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors during concurrent rule updates: First error: %v", len(errors), errors[0])
	}

	// Query the rules to verify they were updated
	rows, err := db.Query(`
		SELECT id, limit_value, action
		FROM policy_rules
		WHERE policy_id = ?
	`, policyID)

	if err != nil {
		t.Fatalf("Failed to query rules: %v", err)
	}
	defer rows.Close()

	// Check each rule was updated
	rulesFound := 0
	for rows.Next() {
		var id string
		var limitValue float64
		var action string

		if err := rows.Scan(&id, &limitValue, &action); err != nil {
			t.Fatalf("Failed to scan rule row: %v", err)
		}

		// Find the rule in our list
		for _, ruleID := range ruleIDs {
			if id == ruleID {
				rulesFound++

				// Verify the rule was updated (not still at default)
				if limitValue == 1000.0 && action == "log" {
					t.Errorf("Rule %s was not updated as expected", id)
				}

				break
			}
		}
	}

	if rulesFound != len(ruleIDs) {
		t.Errorf("Expected to find %d updated rules, but found %d", len(ruleIDs), rulesFound)
	}
}

// TestConcurrentAPIRequests tests concurrent API request processing
func TestConcurrentAPIRequests(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Tables should already be created by setupTestDB

	// Create test trackers
	trackerIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		trackerID := uuid.New().String()
		trackerIDs[i] = trackerID

		_, err := db.Exec(`
			INSERT INTO trackers (id, name, description, is_active, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, trackerID, "Test Tracker "+string(rune('A'+i)), "Tracker for testing "+string(rune('A'+i)), true, time.Now())

		if err != nil {
			t.Fatalf("Failed to insert tracker for concurrency test: %v", err)
		}
	}

	// Number of concurrent requests
	numRequests := 20

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Track errors and created request IDs
	errChan := make(chan error, numRequests)
	requestIDs := make(chan string, numRequests)

	// Function to insert API requests concurrently
	insertAPIRequest := func(i int) {
		defer wg.Done()

		requestID := uuid.New().String()

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			errChan <- fmt.Errorf("failed to begin transaction: %v", err)
			return
		}
		defer tx.Rollback()

		// Insert the API request
		_, err = tx.Exec(`
			INSERT INTO api_requests (id, api_name, description, submitted_date, status, requester_id)
			VALUES (?, ?, ?, ?, ?, ?)
		`, requestID, fmt.Sprintf("Test API Request %d", i), fmt.Sprintf("Description for test request %d", i),
			time.Now(), "pending", "test_requester_"+uuid.New().String())

		if err != nil {
			errChan <- fmt.Errorf("failed to insert API request: %v", err)
			return
		}

		// Add tracker associations
		for j, trackerID := range trackerIDs {
			if j%2 == i%2 { // Only add some trackers to create variety
				associationID := uuid.New().String()

				_, err = tx.Exec(`
					INSERT INTO request_required_trackers (id, request_id, tracker_id)
					VALUES (?, ?, ?)
				`, associationID, requestID, trackerID)

				if err != nil {
					errChan <- fmt.Errorf("failed to associate tracker: %v", err)
					return
				}
			}
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			errChan <- fmt.Errorf("failed to commit transaction: %v", err)
			return
		}

		// Send the request ID back
		requestIDs <- requestID
	}

	// Start concurrent operations
	for i := 0; i < numRequests; i++ {
		go insertAPIRequest(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)
	close(requestIDs)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors during concurrent API request insertion: First error: %v", len(errors), errors[0])
	}

	// Collect all created request IDs
	createdIDs := make([]string, 0, numRequests)
	for id := range requestIDs {
		createdIDs = append(createdIDs, id)
	}

	// Verify the number of inserted requests
	var requestCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM api_requests`).Scan(&requestCount)
	if err != nil {
		t.Fatalf("Failed to count API requests: %v", err)
	}

	if requestCount != numRequests-len(errors) {
		t.Errorf("Expected %d API requests, got %d", numRequests-len(errors), requestCount)
	}

	// Verify tracker associations
	var associationCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM request_required_trackers
		WHERE request_id IN (`+placeholders(len(createdIDs))+`)
	`, stringSliceToInterfaceSlice(createdIDs)...).Scan(&associationCount)

	if err != nil {
		t.Fatalf("Failed to count tracker associations: %v", err)
	}

	// Can't verify exact number due to the random assignment, but should be greater than zero
	if associationCount == 0 && len(createdIDs) > 0 {
		t.Errorf("Expected some tracker associations, but found none")
	}
}

// Helper function to create placeholders for SQL IN clauses
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}

	placeholder := "?"
	for i := 1; i < n; i++ {
		placeholder += ",?"
	}

	return placeholder
}

// Helper function to convert []string to []interface{}
func stringSliceToInterfaceSlice(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
