package db

import (
	"database/sql"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"os"
	"sync"
	"testing"
	"time"
)

// TestConcurrentUsageTracking tests concurrent usage tracking in the api_usage table
func TestConcurrentUsageTracking(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	// Run all migrations
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run base migrations: %v", err)
	}
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run API management migrations: %v", err)
	}

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
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	// Run all migrations
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run base migrations: %v", err)
	}
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run API management migrations: %v", err)
	}

	// Create a test policy for concurrent rule updates
	policyID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, policyID, "Concurrent Rules Policy", "Policy for concurrent rule testing", "composite", true, time.Now(), time.Now(), "test_user")

	if err != nil {
		t.Fatalf("Failed to insert policy for concurrency test: %v", err)
	}

	// Create initial rules
	ruleTypes := []string{"token", "time", "credit", "rate"}
	ruleIDs := make([]string, len(ruleTypes))

	for i, ruleType := range ruleTypes {
		ruleID := uuid.New().String()
		ruleIDs[i] = ruleID

		_, err := db.Exec(`
			INSERT INTO policy_rules (id, policy_id, rule_type, limit_value, period, action, priority)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, ruleID, policyID, ruleType, 1000.0, "day", "log", 100)

		if err != nil {
			t.Fatalf("Failed to insert initial rule: %v", err)
		}
	}

	// Number of concurrent updates per rule
	updatesPerRule := 5

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(len(ruleIDs) * updatesPerRule)

	// Track errors from goroutines
	errChan := make(chan error, len(ruleIDs)*updatesPerRule)

	// Function to update rule concurrently
	updateRule := func(ruleID string, limitValue float64, action string) {
		defer wg.Done()

		_, err := db.Exec(`
			UPDATE policy_rules
			SET limit_value = ?, action = ?
			WHERE id = ?
		`, limitValue, action, ruleID)

		if err != nil {
			errChan <- err
		}
	}

	// Start concurrent updates
	actions := []string{"log", "notify", "throttle", "block"}
	for _, ruleID := range ruleIDs {
		for i := 0; i < updatesPerRule; i++ {
			// Vary the limit value and action for each update
			limitValue := 1000.0 + float64(i*100)
			action := actions[i%len(actions)]
			go updateRule(ruleID, limitValue, action)
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
		t.Errorf("Got %d errors during concurrent rule updates: First error: %v", len(errors), errors[0])
	}

	// Verify that all rules were updated
	rows, err := db.Query(`SELECT id, limit_value, action FROM policy_rules WHERE policy_id = ?`, policyID)
	if err != nil {
		t.Fatalf("Failed to query rules: %v", err)
	}
	defer rows.Close()

	// Each rule should have a non-default value
	rulesFound := 0
	for rows.Next() {
		var id string
		var limitValue float64
		var action string

		if err := rows.Scan(&id, &limitValue, &action); err != nil {
			t.Fatalf("Failed to scan rule data: %v", err)
		}

		// Check if this is one of our test rules
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
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	// Run all migrations
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run base migrations: %v", err)
	}
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run API management migrations: %v", err)
	}

	// Create a test tracker for request associations
	trackerID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO trackers (id, name, description, is_active)
		VALUES (?, ?, ?, ?)
	`, trackerID, "Concurrent Test Tracker", "Tracker for concurrent testing", true)

	if err != nil {
		t.Fatalf("Failed to insert tracker for concurrency test: %v", err)
	}

	// Number of concurrent requests to process
	requestCount := 20

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(requestCount * 2) // Each request has an insert and an update

	// Track errors from goroutines
	errChan := make(chan error, requestCount*2)

	// Track request IDs for verification
	requestIDs := make([]string, requestCount)

	// Function to create and process API requests concurrently
	processRequest := func(index int) {
		defer wg.Done() // For the request insert

		// Create request ID and store it for verification
		requestID := uuid.New().String()
		requestIDs[index] = requestID

		// Insert API request
		_, err := db.Exec(`
			INSERT INTO api_requests (id, api_name, description, status, requester_id)
			VALUES (?, ?, ?, ?, ?)
		`, requestID, "Test API "+uuid.New().String()[:8], "Concurrent request test", "pending", "test_user_"+uuid.New().String()[:8])

		if err != nil {
			errChan <- err
			return
		}

		// Associate tracker with request
		_, err = db.Exec(`
			INSERT INTO request_required_trackers (id, request_id, tracker_id)
			VALUES (?, ?, ?)
		`, uuid.New().String(), requestID, trackerID)

		if err != nil {
			errChan <- err
			return
		}

		// Start a goroutine to update the request status
		go func() {
			defer wg.Done() // For the request update

			// Small delay to simulate processing time
			time.Sleep(time.Millisecond * time.Duration(10+index%10))

			// Update request status
			_, err := db.Exec(`
				UPDATE api_requests
				SET status = ?, approved_date = ?
				WHERE id = ?
			`, "approved", time.Now(), requestID)

			if err != nil {
				errChan <- err
			}
		}()
	}

	// Start concurrent request processing
	for i := 0; i < requestCount; i++ {
		go processRequest(i)
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
		t.Errorf("Got %d errors during concurrent request processing: First error: %v", len(errors), errors[0])
	}

	// Verify that all requests were inserted and updated
	for i, requestID := range requestIDs {
		var status string
		var approvedDate sql.NullTime

		err := db.QueryRow(`
			SELECT status, approved_date
			FROM api_requests
			WHERE id = ?
		`, requestID).Scan(&status, &approvedDate)

		if err != nil {
			t.Errorf("Failed to fetch request %d (%s): %v", i, requestID, err)
			continue
		}

		if status != "approved" {
			t.Errorf("Request %d (%s) status = %s, expected 'approved'", i, requestID, status)
		}

		if !approvedDate.Valid {
			t.Errorf("Request %d (%s) approved_date is NULL, expected a timestamp", i, requestID)
		}

		// Verify tracker association
		var associationCount int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM request_required_trackers
			WHERE request_id = ? AND tracker_id = ?
		`, requestID, trackerID).Scan(&associationCount)

		if err != nil {
			t.Errorf("Failed to check tracker association for request %d (%s): %v", i, requestID, err)
		}

		if associationCount != 1 {
			t.Errorf("Expected 1 tracker association for request %d (%s), found %d", i, requestID, associationCount)
		}
	}
}
