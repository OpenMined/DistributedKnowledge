package db

import (
	"github.com/google/uuid"
	"os"
	"testing"
	"time"
)

// TestPolicyManagementCRUD tests the CRUD operations for policies and policy rules
func TestPolicyManagementCRUD(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Step 1: Test Free Policy Creation
	t.Run("CreateFreePolicy", func(t *testing.T) {
		policyID := uuid.New().String()
		now := time.Now()

		policy := &Policy{
			ID:          policyID,
			Name:        "Free Policy",
			Description: "Unlimited access without restrictions",
			Type:        "free",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		err := CreatePolicy(db, policy)
		if err != nil {
			t.Fatalf("Failed to create free policy: %v", err)
		}

		// Retrieve the policy
		retrievedPolicy, err := GetPolicy(db, policyID)
		if err != nil {
			t.Fatalf("Failed to retrieve policy: %v", err)
		}

		if retrievedPolicy.Name != "Free Policy" {
			t.Errorf("Expected policy name 'Free Policy', got '%s'", retrievedPolicy.Name)
		}

		if retrievedPolicy.Type != "free" {
			t.Errorf("Expected policy type 'free', got '%s'", retrievedPolicy.Type)
		}
	})

	// Step 2: Test Rate-Limited Policy Creation with Rules
	t.Run("CreateRateLimitedPolicy", func(t *testing.T) {
		policyID := uuid.New().String()
		now := time.Now()

		// Create policy
		policy := &Policy{
			ID:          policyID,
			Name:        "Rate Limited Policy",
			Description: "100 calls per hour",
			Type:        "rate",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to start transaction: %v", err)
		}

		// Create policy in transaction
		err = CreatePolicyTx(tx, policy)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create rate limited policy: %v", err)
		}

		// Create a rule for the policy
		rule := &PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policyID,
			RuleType:   "rate",
			LimitValue: 100,
			Period:     "hour",
			Action:     "block",
			Priority:   100,
			CreatedAt:  now,
		}

		err = CreatePolicyRuleTx(tx, rule)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create policy rule: %v", err)
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify the policy and rule were created
		policyWithRules, err := GetPolicyWithRules(db, policyID)
		if err != nil {
			t.Fatalf("Failed to retrieve policy with rules: %v", err)
		}

		if len(policyWithRules.Rules) != 1 {
			t.Fatalf("Expected 1 rule, got %d", len(policyWithRules.Rules))
		}

		rule = &policyWithRules.Rules[0]
		if rule.RuleType != "rate" {
			t.Errorf("Expected rule type 'rate', got '%s'", rule.RuleType)
		}

		if rule.LimitValue != 100 {
			t.Errorf("Expected limit value 100, got %f", rule.LimitValue)
		}

		if rule.Period != "hour" {
			t.Errorf("Expected period 'hour', got '%s'", rule.Period)
		}

		if rule.Action != "block" {
			t.Errorf("Expected action 'block', got '%s'", rule.Action)
		}
	})

	// Step 3: Test Composite Policy Creation with Multiple Rules
	t.Run("CreateCompositePolicy", func(t *testing.T) {
		policyID := uuid.New().String()
		now := time.Now()

		// Create policy
		policy := &Policy{
			ID:          policyID,
			Name:        "Composite Policy",
			Description: "Multiple types of limits",
			Type:        "composite",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to start transaction: %v", err)
		}

		// Create policy in transaction
		err = CreatePolicyTx(tx, policy)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create composite policy: %v", err)
		}

		// Create rate limit rule
		rateRule := &PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policyID,
			RuleType:   "rate",
			LimitValue: 1000,
			Period:     "day",
			Action:     "notify",
			Priority:   100,
			CreatedAt:  now,
		}

		err = CreatePolicyRuleTx(tx, rateRule)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create rate rule: %v", err)
		}

		// Create token limit rule
		tokenRule := &PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policyID,
			RuleType:   "token",
			LimitValue: 100000,
			Period:     "month",
			Action:     "throttle",
			Priority:   200,
			CreatedAt:  now,
		}

		err = CreatePolicyRuleTx(tx, tokenRule)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create token rule: %v", err)
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify the policy and rules were created
		policyWithRules, err := GetPolicyWithRules(db, policyID)
		if err != nil {
			t.Fatalf("Failed to retrieve policy with rules: %v", err)
		}

		if len(policyWithRules.Rules) != 2 {
			t.Fatalf("Expected 2 rules, got %d", len(policyWithRules.Rules))
		}

		// Check that the rules match our expectations
		foundRateRule := false
		foundTokenRule := false

		for _, rule := range policyWithRules.Rules {
			if rule.RuleType == "rate" && rule.LimitValue == 1000 && rule.Period == "day" {
				foundRateRule = true
			} else if rule.RuleType == "token" && rule.LimitValue == 100000 && rule.Period == "month" {
				foundTokenRule = true
			}
		}

		if !foundRateRule {
			t.Errorf("Could not find the expected rate rule")
		}

		if !foundTokenRule {
			t.Errorf("Could not find the expected token rule")
		}
	})

	// Step 4: Test Policy Update
	t.Run("UpdatePolicy", func(t *testing.T) {
		// Create a policy to update
		policyID := uuid.New().String()
		now := time.Now()

		policy := &Policy{
			ID:          policyID,
			Name:        "Policy To Update",
			Description: "This will be updated",
			Type:        "time",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		err := CreatePolicy(db, policy)
		if err != nil {
			t.Fatalf("Failed to create policy for update test: %v", err)
		}

		// Update the policy
		policy.Name = "Updated Policy Name"
		policy.Description = "This was updated"
		policy.IsActive = false
		policy.UpdatedAt = time.Now()

		err = UpdatePolicy(db, policy)
		if err != nil {
			t.Fatalf("Failed to update policy: %v", err)
		}

		// Verify the update
		updatedPolicy, err := GetPolicy(db, policyID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated policy: %v", err)
		}

		if updatedPolicy.Name != "Updated Policy Name" {
			t.Errorf("Update failed: expected name 'Updated Policy Name', got '%s'", updatedPolicy.Name)
		}

		if updatedPolicy.Description != "This was updated" {
			t.Errorf("Update failed: expected description 'This was updated', got '%s'", updatedPolicy.Description)
		}

		if updatedPolicy.IsActive {
			t.Errorf("Update failed: expected IsActive to be false")
		}
	})

	// Step 5: Test Policy Rules Update
	t.Run("UpdatePolicyRules", func(t *testing.T) {
		// Create a policy with a rule
		policyID := uuid.New().String()
		now := time.Now()

		// Create policy
		policy := &Policy{
			ID:          policyID,
			Name:        "Policy With Rules To Update",
			Description: "Rules will be updated",
			Type:        "credit",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to start transaction: %v", err)
		}

		// Create policy in transaction
		err = CreatePolicyTx(tx, policy)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create policy for rule update test: %v", err)
		}

		// Create a rule for the policy
		oldRule := &PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policyID,
			RuleType:   "credit",
			LimitValue: 1000,
			Period:     "month",
			Action:     "notify",
			Priority:   100,
			CreatedAt:  now,
		}

		err = CreatePolicyRuleTx(tx, oldRule)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create policy rule for update test: %v", err)
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Create a new transaction for the update
		tx, err = db.Begin()
		if err != nil {
			t.Fatalf("Failed to start transaction for update: %v", err)
		}

		// Delete existing rules
		err = DeletePolicyRulesTx(tx, policyID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to delete existing rules: %v", err)
		}

		// Create a new rule with different values
		newRule := &PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policyID,
			RuleType:   "credit",
			LimitValue: 2000,    // increased from 1000
			Period:     "year",  // changed from month to year
			Action:     "block", // changed from notify to block
			Priority:   50,      // changed from 100 to 50
			CreatedAt:  time.Now(),
		}

		err = CreatePolicyRuleTx(tx, newRule)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to create new policy rule: %v", err)
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit rule update transaction: %v", err)
		}

		// Verify the rule was updated
		policyWithRules, err := GetPolicyWithRules(db, policyID)
		if err != nil {
			t.Fatalf("Failed to retrieve policy with updated rules: %v", err)
		}

		if len(policyWithRules.Rules) != 1 {
			t.Fatalf("Expected 1 rule after update, got %d", len(policyWithRules.Rules))
		}

		updatedRule := policyWithRules.Rules[0]
		if updatedRule.LimitValue != 2000 {
			t.Errorf("Rule update failed: expected limit value 2000, got %f", updatedRule.LimitValue)
		}

		if updatedRule.Period != "year" {
			t.Errorf("Rule update failed: expected period 'year', got '%s'", updatedRule.Period)
		}

		if updatedRule.Action != "block" {
			t.Errorf("Rule update failed: expected action 'block', got '%s'", updatedRule.Action)
		}

		if updatedRule.Priority != 50 {
			t.Errorf("Rule update failed: expected priority 50, got %d", updatedRule.Priority)
		}
	})

	// Step 6: Test PolicyListFiltering
	t.Run("PolicyListFiltering", func(t *testing.T) {
		// Create several policies with different types and statuses
		policies := []struct {
			ID        string
			Name      string
			Type      string
			IsActive  bool
			CreatedBy string
		}{
			{uuid.New().String(), "Active Free Policy", "free", true, "test_user"},
			{uuid.New().String(), "Inactive Rate Policy", "rate", false, "test_user"},
			{uuid.New().String(), "Active Token Policy", "token", true, "another_user"},
			{uuid.New().String(), "Active Time Policy", "time", true, "test_user"},
			{uuid.New().String(), "Inactive Credit Policy", "credit", false, "another_user"},
		}

		// Create all policies
		for _, p := range policies {
			policy := &Policy{
				ID:          p.ID,
				Name:        p.Name,
				Description: "Test policy for filtering",
				Type:        p.Type,
				IsActive:    p.IsActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				CreatedBy:   p.CreatedBy,
			}

			err := CreatePolicy(db, policy)
			if err != nil {
				t.Fatalf("Failed to create policy for filtering test: %v", err)
			}
		}

		// Test 1: Filter by type
		results, total, err := ListPolicies(db, "token", false, "", 10, 0, "created_at", "desc")
		if err != nil {
			t.Fatalf("Failed to filter policies by type: %v", err)
		}

		// At least find the token policy we just created
		if total < 1 {
			t.Errorf("Expected at least 1 token policy, got %d", total)
		}

		foundToken := false
		for _, p := range results {
			if p.Type == "token" {
				foundToken = true
			}
		}

		if !foundToken {
			t.Errorf("Type filter failed: did not find any token policy")
		}

		// Test 2: Filter by active status
		results, total, err = ListPolicies(db, "", true, "", 10, 0, "created_at", "desc")
		if err != nil {
			t.Fatalf("Failed to filter policies by active status: %v", err)
		}

		// Count the active policies from our test set
		activePoliciesCount := 0
		for _, p := range policies {
			if p.IsActive {
				activePoliciesCount++
			}
		}

		// Verify we have at least the active policies we created
		if total < activePoliciesCount {
			t.Errorf("Expected at least %d active policies, got %d", activePoliciesCount, total)
		}

		// Verify all results are active
		for _, p := range results {
			if !p.IsActive {
				t.Errorf("Active filter failed: got inactive policy '%s'", p.Name)
			}
		}

		// Test 3: Filter by created_by
		results, total, err = ListPolicies(db, "", false, "another_user", 10, 0, "created_at", "desc")
		if err != nil {
			t.Fatalf("Failed to filter policies by created_by: %v", err)
		}

		// Count the policies created by "another_user" from our test set
		anotherUserPoliciesCount := 0
		for _, p := range policies {
			if p.CreatedBy == "another_user" {
				anotherUserPoliciesCount++
			}
		}

		// Verify we have the policies created by "another_user"
		if total != anotherUserPoliciesCount {
			t.Errorf("Expected %d policies by 'another_user', got %d", anotherUserPoliciesCount, total)
		}

		// Verify all results are by another_user
		for _, p := range results {
			if p.CreatedBy != "another_user" {
				t.Errorf("Creator filter failed: expected 'another_user', got '%s'", p.CreatedBy)
			}
		}

		// Test 4: Combined filters
		// Count how many active time policies by test_user we created
		activeTimePoliciesByTestUser := 0
		for _, p := range policies {
			if p.Type == "time" && p.IsActive && p.CreatedBy == "test_user" {
				activeTimePoliciesByTestUser++
			}
		}

		// If we did create such a policy, test the combined filter
		if activeTimePoliciesByTestUser > 0 {
			results, total, err = ListPolicies(db, "time", true, "test_user", 10, 0, "created_at", "desc")
			if err != nil {
				t.Fatalf("Failed to apply combined filters: %v", err)
			}

			// Verify we have the expected number of matches
			if total != activeTimePoliciesByTestUser {
				t.Errorf("Expected %d active time policies by test_user, got %d", activeTimePoliciesByTestUser, total)
			}

			// Verify the properties of the matched policies
			for _, p := range results {
				if p.Type != "time" {
					t.Errorf("Combined filter failed: expected type 'time', got '%s'", p.Type)
				}
				if !p.IsActive {
					t.Errorf("Combined filter failed: expected active policy for '%s'", p.Name)
				}
				if p.CreatedBy != "test_user" {
					t.Errorf("Combined filter failed: expected creator 'test_user', got '%s'", p.CreatedBy)
				}
			}
		}
	})

	// Step 7: Test Policy Deletion
	t.Run("DeletePolicy", func(t *testing.T) {
		// Create a policy to delete
		policyID := uuid.New().String()
		now := time.Now()

		policy := &Policy{
			ID:          policyID,
			Name:        "Policy To Delete",
			Description: "This will be deleted",
			Type:        "free",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		err := CreatePolicy(db, policy)
		if err != nil {
			t.Fatalf("Failed to create policy for deletion test: %v", err)
		}

		// Create a rule for the policy to ensure cascade delete works
		rule := &PolicyRule{
			ID:        uuid.New().String(),
			PolicyID:  policyID,
			RuleType:  "free",
			Action:    "log",
			Priority:  100,
			CreatedAt: now,
		}

		err = CreatePolicyRule(db, rule)
		if err != nil {
			t.Fatalf("Failed to create policy rule for deletion test: %v", err)
		}

		// Delete the policy
		err = DeletePolicy(db, policyID)
		if err != nil {
			t.Fatalf("Failed to delete policy: %v", err)
		}

		// Verify the policy was deleted
		_, err = GetPolicy(db, policyID)
		if err == nil {
			t.Errorf("Expected error when retrieving deleted policy, got nil")
		} else if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		// Verify the rules were also deleted (cascade delete)
		rules, err := GetPolicyRules(db, policyID)
		if err != nil {
			t.Fatalf("Failed to check rule deletion: %v", err)
		}

		if len(rules) > 0 {
			t.Errorf("Expected 0 rules after policy deletion, got %d", len(rules))
		}
	})
}

// TestPolicyChangeHistory tests the policy change history functionality
func TestPolicyChangeHistory(t *testing.T) {
	// Skip this test if we're in CI or just running quick tests
	if os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test due to SKIP_DB_TESTS environment variable")
	}

	// Setup test database
	db := setupTestDB(t)
	// Don't close the shared database connection

	// Step 1: Create test policies and API
	t.Run("Setup", func(t *testing.T) {
		// Create two test policies
		oldPolicyID := uuid.New().String()
		newPolicyID := uuid.New().String()
		apiID := uuid.New().String()
		now := time.Now()

		// Create old policy
		oldPolicy := &Policy{
			ID:          oldPolicyID,
			Name:        "Old Policy",
			Description: "Original policy",
			Type:        "free",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		err := CreatePolicy(db, oldPolicy)
		if err != nil {
			t.Fatalf("Failed to create old policy: %v", err)
		}

		// Create new policy
		newPolicy := &Policy{
			ID:          newPolicyID,
			Name:        "New Policy",
			Description: "Replacement policy",
			Type:        "rate",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   "test_user",
		}

		err = CreatePolicy(db, newPolicy)
		if err != nil {
			t.Fatalf("Failed to create new policy: %v", err)
		}

		// Create test API
		_, err = db.Exec(`
			INSERT INTO apis (id, name, description, created_at, updated_at, is_active, api_key, host_user_id, policy_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, apiID, "Test API", "API for testing", now, now, true, "test_key", "test_host", oldPolicyID)

		if err != nil {
			t.Fatalf("Failed to create test API: %v", err)
		}

		// Step 2: Test recording policy changes

		// Create first change record
		firstChangeID := uuid.New().String()

		// Change effective immediately
		effectiveNow := time.Now()
		firstChange := &PolicyChange{
			ID:            firstChangeID,
			APIID:         apiID,
			OldPolicyID:   &oldPolicyID,
			NewPolicyID:   &newPolicyID,
			ChangedAt:     time.Now(),
			ChangedBy:     "test_user",
			EffectiveDate: &effectiveNow,
			ChangeReason:  "Testing immediate change",
		}

		err = CreatePolicyChange(db, firstChange)
		if err != nil {
			t.Fatalf("Failed to create policy change record: %v", err)
		}

		// Update the API policy directly to reflect immediate change
		// (in a real system this would be done automatically)
		_, err = db.Exec("UPDATE apis SET policy_id = ? WHERE id = ?", newPolicyID, apiID)
		if err != nil {
			t.Fatalf("Failed to update API policy after immediate change: %v", err)
		}

		// Create second change record for future scheduling
		secondChangeID := uuid.New().String()
		futureDate := time.Now().Add(24 * time.Hour) // tomorrow

		secondChange := &PolicyChange{
			ID:            secondChangeID,
			APIID:         apiID,
			OldPolicyID:   &newPolicyID,
			NewPolicyID:   &oldPolicyID, // change back to original
			ChangedAt:     time.Now(),
			ChangedBy:     "test_user",
			EffectiveDate: &futureDate,
			ChangeReason:  "Testing scheduled change",
		}

		err = CreatePolicyChange(db, secondChange)
		if err != nil {
			t.Fatalf("Failed to create scheduled policy change record: %v", err)
		}

		// Step 3: Test retrieving policy changes
		changes, err := GetPolicyChangeHistory(db, apiID)
		if err != nil {
			t.Fatalf("Failed to retrieve policy change history: %v", err)
		}

		if len(changes) != 2 {
			t.Fatalf("Expected 2 changes in history, got %d", len(changes))
		}

		// Verify changes are in the correct order (newest first)
		if changes[0].ChangedAt.Before(changes[1].ChangedAt) {
			t.Errorf("Changes not sorted by ChangedAt DESC")
		}

		// Check the content of the changes
		var foundImmediate, foundScheduled bool

		for _, change := range changes {
			if change.ID == firstChangeID {
				foundImmediate = true
				if *change.OldPolicyID != oldPolicyID {
					t.Errorf("Expected old policy ID %s, got %s", oldPolicyID, *change.OldPolicyID)
				}
				if *change.NewPolicyID != newPolicyID {
					t.Errorf("Expected new policy ID %s, got %s", newPolicyID, *change.NewPolicyID)
				}
				if change.ChangeReason != "Testing immediate change" {
					t.Errorf("Expected reason 'Testing immediate change', got '%s'", change.ChangeReason)
				}
			} else if change.ID == secondChangeID {
				foundScheduled = true
				if *change.OldPolicyID != newPolicyID {
					t.Errorf("Expected old policy ID %s, got %s", newPolicyID, *change.OldPolicyID)
				}
				if *change.NewPolicyID != oldPolicyID {
					t.Errorf("Expected new policy ID %s, got %s", oldPolicyID, *change.NewPolicyID)
				}
				if change.ChangeReason != "Testing scheduled change" {
					t.Errorf("Expected reason 'Testing scheduled change', got '%s'", change.ChangeReason)
				}
			}
		}

		if !foundImmediate {
			t.Errorf("Could not find the immediate change record in history")
		}

		if !foundScheduled {
			t.Errorf("Could not find the scheduled change record in history")
		}

		// Verify the API has the first policy applied
		var currentPolicyID string
		err = db.QueryRow("SELECT policy_id FROM apis WHERE id = ?", apiID).Scan(&currentPolicyID)
		if err != nil {
			t.Fatalf("Failed to get current API policy: %v", err)
		}

		if currentPolicyID != newPolicyID {
			t.Errorf("Expected API to have policy %s after immediate change, got %s", newPolicyID, currentPolicyID)
		}

		// Step 4: Test applying a pending policy change

		// Test GetPendingPolicyChanges - there should be none since the scheduled change is for tomorrow
		pendingChanges, err := GetPendingPolicyChanges(db)
		if err != nil {
			t.Fatalf("Failed to get pending policy changes: %v", err)
		}

		// There should be no pending changes yet (all are scheduled for the future)
		if len(pendingChanges) != 0 {
			t.Logf("Note: Found %d pending changes when expecting 0 (this can happen if previous tests created pending changes)", len(pendingChanges))
		}

		// Update the effective date to the past to make it "pending"
		pastDate := time.Now().Add(-1 * time.Hour) // one hour ago
		_, err = db.Exec("UPDATE policy_changes SET effective_date = ? WHERE id = ?", pastDate, secondChangeID)
		if err != nil {
			t.Fatalf("Failed to update effective date: %v", err)
		}

		// Now there should be a pending change
		pendingChanges, err = GetPendingPolicyChanges(db)
		if err != nil {
			t.Fatalf("Failed to get pending policy changes after date update: %v", err)
		}

		pendingForTestAPI := 0
		var pendingChange *PolicyChange
		for _, change := range pendingChanges {
			if change.APIID == apiID {
				pendingForTestAPI++
				pendingChange = change
			}
		}

		if pendingForTestAPI != 1 {
			t.Fatalf("Expected 1 pending change for our test API, got %d", pendingForTestAPI)
		}

		// Apply the pending change
		err = ApplyPendingPolicyChange(db, pendingChange)
		if err != nil {
			t.Fatalf("Failed to apply pending policy change: %v", err)
		}

		// Verify the API now has the old policy again
		err = db.QueryRow("SELECT policy_id FROM apis WHERE id = ?", apiID).Scan(&currentPolicyID)
		if err != nil {
			t.Fatalf("Failed to get current API policy after applying change: %v", err)
		}

		if currentPolicyID != oldPolicyID {
			t.Errorf("Expected API to have policy %s after applying pending change, got %s", oldPolicyID, currentPolicyID)
		}

		// Note: The current implementation of ApplyPendingPolicyChange doesn't delete the applied changes
		// In a real system, we would want to delete or mark the changes as applied
	})
}
