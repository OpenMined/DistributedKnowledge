package http

import (
	"bytes"
	"context"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Helper function for setting up test API keys
func randomAPIKey() string {
	return uuid.New().String()
}

func TestPolicyHandlers(t *testing.T) {
	// Setup test context with database
	ctx, testDB, err := setupTestContext(t)
	if err != nil {
		t.Fatalf("Failed to setup test context: %v", err)
	}
	defer testDB.Close()

	// Set a fixed user ID for testing
	ctx = context.WithValue(ctx, utils.UserIDContextKey, "test-user")

	// Helper function to create a test policy in the database
	createTestPolicy := func(policyType string, isActive bool) (*db.Policy, error) {
		policy := &db.Policy{
			ID:          uuid.New().String(),
			Name:        "Test " + policyType + " Policy",
			Description: "Test policy for " + policyType,
			Type:        policyType,
			IsActive:    isActive,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			CreatedBy:   "test-user",
		}

		dbInst, err := utils.DBFromContext(ctx)
		if err != nil {
			return nil, err
		}

		// Start transaction
		tx, err := dbInst.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		// Create policy
		err = db.CreatePolicyTx(tx, policy)
		if err != nil {
			return nil, err
		}

		// For non-free policies, add a rule
		if policyType != "free" {
			rule := &db.PolicyRule{
				ID:         uuid.New().String(),
				PolicyID:   policy.ID,
				RuleType:   policyType,
				LimitValue: 100,
				Period:     "day",
				Action:     "block",
				Priority:   100,
				CreatedAt:  time.Now(),
			}

			err = db.CreatePolicyRuleTx(tx, rule)
			if err != nil {
				return nil, err
			}
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return nil, err
		}

		return policy, nil
	}

	// Helper function to create a test API with a policy
	createTestAPI := func(policyID *string) (*db.API, error) {
		api := &db.API{
			ID:          uuid.New().String(),
			Name:        "Test API",
			Description: "Test API Description",
			IsActive:    true,
			APIKey:      randomAPIKey(), // Use a unique API key to avoid conflicts
			HostUserID:  "test-user",
			PolicyID:    policyID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		dbInst, err := utils.DBFromContext(ctx)
		if err != nil {
			return nil, err
		}

		err = db.CreateAPI(dbInst, api)
		if err != nil {
			return nil, err
		}

		return api, nil
	}

	t.Run("HandleListPolicies", func(t *testing.T) {
		// Create a few test policies of different types
		_, err := createTestPolicy("free", true)
		if err != nil {
			t.Fatalf("Failed to create free policy: %v", err)
		}

		_, err = createTestPolicy("rate", true)
		if err != nil {
			t.Fatalf("Failed to create rate policy: %v", err)
		}

		_, err = createTestPolicy("token", false)
		if err != nil {
			t.Fatalf("Failed to create token policy: %v", err)
		}

		// Test 1: List all policies
		req := httptest.NewRequest(http.MethodGet, "/api/policies", nil)
		rec := httptest.NewRecorder()

		HandleListPolicies(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response PolicyListResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// We should have at least 2 policies (the ones we just created that are active)
		if response.Total < 2 {
			t.Errorf("Expected at least 2 policies, got %d", response.Total)
		}

		// Test 2: Filter by policy type
		req = httptest.NewRequest(http.MethodGet, "/api/policies?type=free", nil)
		rec = httptest.NewRecorder()

		HandleListPolicies(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check that we have free policies
		foundFreePolicy := false
		for _, policy := range response.Policies {
			if policy.Type == "free" {
				foundFreePolicy = true
			}
			// Ensure other types aren't included
			if policy.Type != "free" {
				t.Errorf("Expected only free policy types, found %s", policy.Type)
			}
		}

		if !foundFreePolicy {
			t.Errorf("Expected to find at least one free policy")
		}

		// Test 3: Filter by active status
		req = httptest.NewRequest(http.MethodGet, "/api/policies?active=false", nil)
		rec = httptest.NewRecorder()

		HandleListPolicies(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check that we have inactive policies (tokenPolicy is inactive)
		foundInactivePolicy := false
		if response.Total > 0 {
			for range response.Policies {
				foundInactivePolicy = true
				break
			}
		}

		if !foundInactivePolicy && response.Total > 0 {
			t.Errorf("Expected to find at least one inactive policy")
		}
	})

	t.Run("HandleGetPolicy", func(t *testing.T) {
		// Create a policy with rules
		tokenPolicy, err := createTestPolicy("token", true)
		if err != nil {
			t.Fatalf("Failed to create token policy: %v", err)
		}

		// Test 1: Get existing policy
		req := httptest.NewRequest(http.MethodGet, "/api/policies/"+tokenPolicy.ID, nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		// Set the path parameter
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": tokenPolicy.ID}))

		HandleGetPolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response PolicyDetail
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != tokenPolicy.ID {
			t.Errorf("Expected policy ID %s, got %s", tokenPolicy.ID, response.ID)
		}

		if response.Type != "token" {
			t.Errorf("Expected policy type 'token', got %s", response.Type)
		}

		if len(response.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(response.Rules))
		}

		// Test 2: Non-existent policy
		req = httptest.NewRequest(http.MethodGet, "/api/policies/non-existent", nil)
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": "non-existent"}))
		rec = httptest.NewRecorder()

		HandleGetPolicy(ctx, rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("HandleCreatePolicy", func(t *testing.T) {
		// Test 1: Create a free policy
		freePolicyReq := CreatePolicyRequest{
			Name:        "New Free Policy",
			Description: "Test policy creation",
			Type:        "free",
		}

		body, err := json.Marshal(freePolicyReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		HandleCreatePolicy(ctx, rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
		}

		var freeResponse PolicyDetail
		err = json.Unmarshal(rec.Body.Bytes(), &freeResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if freeResponse.Name != "New Free Policy" {
			t.Errorf("Expected policy name 'New Free Policy', got '%s'", freeResponse.Name)
		}

		if freeResponse.Type != "free" {
			t.Errorf("Expected policy type 'free', got '%s'", freeResponse.Type)
		}

		// Test 2: Create a policy with rules
		ratePolicyReq := CreatePolicyRequest{
			Name:        "New Rate Policy",
			Description: "Test policy with rules",
			Type:        "rate",
			Rules: []PolicyRule{
				{
					RuleType:   "rate",
					LimitValue: 500,
					Period:     "hour",
					Action:     "block",
					Priority:   100,
				},
			},
		}

		body, err = json.Marshal(ratePolicyReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()

		HandleCreatePolicy(ctx, rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
		}

		var rateResponse PolicyDetail
		err = json.Unmarshal(rec.Body.Bytes(), &rateResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if rateResponse.Name != "New Rate Policy" {
			t.Errorf("Expected policy name 'New Rate Policy', got '%s'", rateResponse.Name)
		}

		if rateResponse.Type != "rate" {
			t.Errorf("Expected policy type 'rate', got '%s'", rateResponse.Type)
		}

		if len(rateResponse.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(rateResponse.Rules))
		} else {
			if rateResponse.Rules[0].Type != "rate" {
				t.Errorf("Expected rule type 'rate', got '%s'", rateResponse.Rules[0].Type)
			}
			if rateResponse.Rules[0].Limit != 500 {
				t.Errorf("Expected rule limit 500, got %f", rateResponse.Rules[0].Limit)
			}
			if rateResponse.Rules[0].Period != "hour" {
				t.Errorf("Expected rule period 'hour', got '%s'", rateResponse.Rules[0].Period)
			}
		}

		// Test 3: Error case - non-free policy without rules
		invalidPolicyReq := CreatePolicyRequest{
			Name:        "Invalid Policy",
			Description: "Policy without required rules",
			Type:        "token",
			Rules:       []PolicyRule{}, // Empty rules, should fail
		}

		body, err = json.Marshal(invalidPolicyReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/policies", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()

		HandleCreatePolicy(ctx, rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("HandleUpdatePolicy", func(t *testing.T) {
		// Create a policy to update
		policy, err := createTestPolicy("token", true)
		if err != nil {
			t.Fatalf("Failed to create policy for update test: %v", err)
		}

		// Test 1: Update policy name and description
		updateReq := UpdatePolicyRequest{
			Name:        stringPtr("Updated Policy Name"),
			Description: stringPtr("Updated description"),
		}

		body, err := json.Marshal(updateReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPatch, "/api/policies/"+policy.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": policy.ID}))
		rec := httptest.NewRecorder()

		HandleUpdatePolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response PolicyDetail
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "Updated Policy Name" {
			t.Errorf("Expected policy name 'Updated Policy Name', got '%s'", response.Name)
		}

		// Test 2: Update policy rules
		updateRulesReq := UpdatePolicyRequest{
			Rules: []PolicyRule{
				{
					RuleType:   "token",
					LimitValue: 1000,
					Period:     "month",
					Action:     "notify",
					Priority:   200,
				},
			},
		}

		body, err = json.Marshal(updateRulesReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req = httptest.NewRequest(http.MethodPatch, "/api/policies/"+policy.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": policy.ID}))
		rec = httptest.NewRecorder()

		HandleUpdatePolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Rules) != 1 {
			t.Errorf("Expected 1 rule, got %d", len(response.Rules))
		} else {
			if response.Rules[0].Limit != 1000 {
				t.Errorf("Expected rule limit 1000, got %f", response.Rules[0].Limit)
			}
			if response.Rules[0].Period != "month" {
				t.Errorf("Expected rule period 'month', got '%s'", response.Rules[0].Period)
			}
			if response.Rules[0].Action != "notify" {
				t.Errorf("Expected rule action 'notify', got '%s'", response.Rules[0].Action)
			}
		}

		// Test 3: Update policy active status
		updateStatusReq := UpdatePolicyRequest{
			IsActive: boolPtr(false),
		}

		body, err = json.Marshal(updateStatusReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req = httptest.NewRequest(http.MethodPatch, "/api/policies/"+policy.ID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": policy.ID}))
		rec = httptest.NewRecorder()

		HandleUpdatePolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// Verify the policy is now inactive
		dbInst, err := utils.DBFromContext(ctx)
		if err != nil {
			t.Fatalf("Failed to get database: %v", err)
		}

		updatedPolicy, err := db.GetPolicy(dbInst, policy.ID)
		if err != nil {
			t.Fatalf("Failed to get updated policy: %v", err)
		}

		if updatedPolicy.IsActive {
			t.Errorf("Expected policy to be inactive, but it's still active")
		}
	})

	t.Run("HandleChangeAPIPolicy", func(t *testing.T) {
		// Create two policies and an API
		freePolicy, err := createTestPolicy("free", true)
		if err != nil {
			t.Fatalf("Failed to create free policy: %v", err)
		}

		ratePolicy, err := createTestPolicy("rate", true)
		if err != nil {
			t.Fatalf("Failed to create rate policy: %v", err)
		}

		// Create API with the free policy
		api, err := createTestAPI(&freePolicy.ID)
		if err != nil {
			t.Fatalf("Failed to create API: %v", err)
		}

		// Test 1: Change policy with immediate effect
		changeReq := ChangePolicyRequest{
			PolicyID:             ratePolicy.ID,
			EffectiveImmediately: true,
			ChangeReason:         "Test immediate change",
		}

		body, err := json.Marshal(changeReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/apis/"+api.ID+"/policy", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": api.ID}))
		rec := httptest.NewRecorder()

		HandleChangeAPIPolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response PolicyChangeResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.APIID != api.ID {
			t.Errorf("Expected API ID %s, got %s", api.ID, response.APIID)
		}

		if response.OldPolicy == nil || response.OldPolicy.ID != freePolicy.ID {
			t.Errorf("Expected old policy ID %s", freePolicy.ID)
		}

		if response.NewPolicy == nil || response.NewPolicy.ID != ratePolicy.ID {
			t.Errorf("Expected new policy ID %s", ratePolicy.ID)
		}

		if response.ChangeReason != "Test immediate change" {
			t.Errorf("Expected change reason 'Test immediate change', got '%s'", response.ChangeReason)
		}

		// Verify the API's policy was actually changed
		dbInst, err := utils.DBFromContext(ctx)
		if err != nil {
			t.Fatalf("Failed to get database: %v", err)
		}

		updatedAPI, err := db.GetAPI(dbInst, api.ID)
		if err != nil {
			t.Fatalf("Failed to get updated API: %v", err)
		}

		if updatedAPI.PolicyID == nil || *updatedAPI.PolicyID != ratePolicy.ID {
			t.Errorf("API policy was not updated as expected")
		}

		// Test 2: Schedule a future policy change
		futureDate := time.Now().Add(24 * time.Hour) // tomorrow
		changeReq = ChangePolicyRequest{
			PolicyID:             freePolicy.ID, // change back to free
			EffectiveImmediately: false,
			ScheduledDate:        &futureDate,
			ChangeReason:         "Test scheduled change",
		}

		body, err = json.Marshal(changeReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/apis/"+api.ID+"/policy", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": api.ID}))
		rec = httptest.NewRecorder()

		HandleChangeAPIPolicy(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ChangeReason != "Test scheduled change" {
			t.Errorf("Expected change reason 'Test scheduled change', got '%s'", response.ChangeReason)
		}

		if response.EffectiveDate == nil || !response.EffectiveDate.After(time.Now()) {
			t.Errorf("Expected future effective date")
		}

		// Verify the API's policy is still the rate policy (hasn't changed yet)
		updatedAPI, err = db.GetAPI(dbInst, api.ID)
		if err != nil {
			t.Fatalf("Failed to get updated API: %v", err)
		}

		if updatedAPI.PolicyID == nil || *updatedAPI.PolicyID != ratePolicy.ID {
			t.Errorf("API policy should not have changed yet")
		}
	})

	t.Run("HandleGetAPIPolicyHistory", func(t *testing.T) {
		// Create policies and API
		freePolicy, err := createTestPolicy("free", true)
		if err != nil {
			t.Fatalf("Failed to create free policy: %v", err)
		}

		ratePolicy, err := createTestPolicy("rate", true)
		if err != nil {
			t.Fatalf("Failed to create rate policy: %v", err)
		}

		tokenPolicy, err := createTestPolicy("token", true)
		if err != nil {
			t.Fatalf("Failed to create token policy: %v", err)
		}

		// Create API without policy
		api, err := createTestAPI(nil)
		if err != nil {
			t.Fatalf("Failed to create API: %v", err)
		}

		// Create some policy changes in the database
		dbInst, err := utils.DBFromContext(ctx)
		if err != nil {
			t.Fatalf("Failed to get database: %v", err)
		}

		// First change
		now := time.Now()
		change1 := &db.PolicyChange{
			ID:            uuid.New().String(),
			APIID:         api.ID,
			NewPolicyID:   &freePolicy.ID,
			ChangedAt:     now.Add(-3 * time.Hour), // 3 hours ago
			ChangedBy:     "test-user",
			EffectiveDate: &now,
			ChangeReason:  "First policy assignment",
		}

		err = db.CreatePolicyChange(dbInst, change1)
		if err != nil {
			t.Fatalf("Failed to create policy change record: %v", err)
		}

		// Second change
		change2 := &db.PolicyChange{
			ID:            uuid.New().String(),
			APIID:         api.ID,
			OldPolicyID:   &freePolicy.ID,
			NewPolicyID:   &ratePolicy.ID,
			ChangedAt:     now.Add(-2 * time.Hour), // 2 hours ago
			ChangedBy:     "test-user",
			EffectiveDate: &now,
			ChangeReason:  "Switch to rate limiting",
		}

		err = db.CreatePolicyChange(dbInst, change2)
		if err != nil {
			t.Fatalf("Failed to create policy change record: %v", err)
		}

		// Third change
		change3 := &db.PolicyChange{
			ID:            uuid.New().String(),
			APIID:         api.ID,
			OldPolicyID:   &ratePolicy.ID,
			NewPolicyID:   &tokenPolicy.ID,
			ChangedAt:     now.Add(-1 * time.Hour), // 1 hour ago
			ChangedBy:     "test-user",
			EffectiveDate: &now,
			ChangeReason:  "Switch to token limiting",
		}

		err = db.CreatePolicyChange(dbInst, change3)
		if err != nil {
			t.Fatalf("Failed to create policy change record: %v", err)
		}

		// Now request the history
		req := httptest.NewRequest(http.MethodGet, "/api/apis/"+api.ID+"/policy/history", nil)
		req = req.WithContext(context.WithValue(req.Context(), PathParamContextKey, map[string]string{"id": api.ID}))
		rec := httptest.NewRecorder()

		HandleGetAPIPolicyHistory(ctx, rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response PolicyChangeHistoryResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.APIID != api.ID {
			t.Errorf("Expected API ID %s, got %s", api.ID, response.APIID)
		}

		if len(response.Changes) != 3 {
			t.Errorf("Expected 3 policy changes, got %d", len(response.Changes))
		}

		// Changes should be in reverse chronological order (newest first)
		if len(response.Changes) >= 3 {
			if response.Changes[0].ChangeReason != "Switch to token limiting" {
				t.Errorf("Expected first change to be 'Switch to token limiting', got '%s'", response.Changes[0].ChangeReason)
			}
			if response.Changes[1].ChangeReason != "Switch to rate limiting" {
				t.Errorf("Expected second change to be 'Switch to rate limiting', got '%s'", response.Changes[1].ChangeReason)
			}
			if response.Changes[2].ChangeReason != "First policy assignment" {
				t.Errorf("Expected third change to be 'First policy assignment', got '%s'", response.Changes[2].ChangeReason)
			}
		}
	})
}

// Helper functions for creating pointer types
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
