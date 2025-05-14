package http

import (
	"context"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"time"
)

// HandleListPolicies handles GET /api/policies
func HandleListPolicies(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	policyType := r.URL.Query().Get("type")

	activeOnly := true // default to only active policies
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		if val, err := strconv.ParseBool(activeStr); err == nil {
			activeOnly = val
		}
	}

	// Parse pagination parameters
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	// Parse sorting parameters
	sort := "created_at" // default
	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		if sortParam == "name" || sortParam == "created_at" || sortParam == "type" {
			sort = sortParam
		}
	}

	order := "desc" // default
	if orderParam := r.URL.Query().Get("order"); orderParam != "" {
		if orderParam == "asc" || orderParam == "desc" {
			order = orderParam
		}
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// List policies
	policies, total, err := db.ListPolicies(database, policyType, activeOnly, currentUserID, limit, offset, sort, order)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve policies: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	policyDetails := make([]PolicyDetail, 0, len(policies))
	for _, policy := range policies {
		// Get policy rules
		rules, err := db.GetPolicyRules(database, policy.ID)
		if err != nil {
			// Log error but continue
			utils.LogError(ctx, "Failed to get rules for policy %s: %v", policy.ID, err)
			rules = []db.PolicyRule{}
		}

		// Convert rules to response format
		ruleDetails := make([]PolicyRuleDetail, 0, len(rules))
		for _, rule := range rules {
			ruleDetail := PolicyRuleDetail{
				Type:   rule.RuleType,
				Limit:  rule.LimitValue,
				Period: rule.Period,
				Action: rule.Action,
			}
			ruleDetails = append(ruleDetails, ruleDetail)
		}

		policyDetail := PolicyDetail{
			ID:    policy.ID,
			Name:  policy.Name,
			Type:  policy.Type,
			Rules: ruleDetails,
		}

		policyDetails = append(policyDetails, policyDetail)
	}

	response := PolicyListResponse{
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		Policies: policyDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetPolicy handles GET /api/policies/:id
func HandleGetPolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get policy ID from path
	policyID := getPathParam(r, "id")
	if policyID == "" {
		sendErrorResponse(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Get policy
	policy, err := db.GetPolicy(database, policyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Policy not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve policy: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Only the creator or local user can view policies
	if policy.CreatedBy != currentUserID && currentUserID != "local-user" {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get policy rules
	rules, err := db.GetPolicyRules(database, policyID)
	if err != nil {
		// This shouldn't prevent returning the policy, but log it
		utils.LogError(ctx, "Failed to get rules for policy %s: %v", policyID, err)
		rules = []db.PolicyRule{}
	}

	// Convert rules to response format
	ruleDetails := make([]PolicyRuleDetail, 0, len(rules))
	for _, rule := range rules {
		ruleDetail := PolicyRuleDetail{
			Type:   rule.RuleType,
			Limit:  rule.LimitValue,
			Period: rule.Period,
			Action: rule.Action,
		}
		ruleDetails = append(ruleDetails, ruleDetail)
	}

	// Create the response
	response := PolicyDetail{
		ID:    policy.ID,
		Name:  policy.Name,
		Type:  policy.Type,
		Rules: ruleDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleCreatePolicy handles POST /api/policies
func HandleCreatePolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		sendErrorResponse(w, "Policy name is required", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		sendErrorResponse(w, "Policy type is required", http.StatusBadRequest)
		return
	}

	// Validate policy type
	validTypes := map[string]bool{
		"free":      true,
		"rate":      true,
		"token":     true,
		"time":      true,
		"credit":    true,
		"composite": true,
	}

	if !validTypes[req.Type] {
		sendErrorResponse(w, "Invalid policy type. Must be one of: free, rate, token, time, credit, composite", http.StatusBadRequest)
		return
	}

	// Validate rules based on policy type
	if req.Type != "free" && len(req.Rules) == 0 {
		sendErrorResponse(w, "Rules are required for non-free policies", http.StatusBadRequest)
		return
	}

	// For non-composite policies, ensure rule types match policy type
	if req.Type != "free" && req.Type != "composite" {
		for _, rule := range req.Rules {
			if rule.RuleType != req.Type {
				sendErrorResponse(w, fmt.Sprintf("Rule type '%s' doesn't match policy type '%s'", rule.RuleType, req.Type), http.StatusBadRequest)
				return
			}
		}
	}

	// Validate each rule
	for i, rule := range req.Rules {
		if rule.RuleType == "" {
			sendErrorResponse(w, fmt.Sprintf("Rule %d is missing rule_type", i+1), http.StatusBadRequest)
			return
		}

		if rule.Action == "" {
			sendErrorResponse(w, fmt.Sprintf("Rule %d is missing action", i+1), http.StatusBadRequest)
			return
		}

		// Validate action
		validActions := map[string]bool{
			"block":    true,
			"throttle": true,
			"notify":   true,
			"log":      true,
		}

		if !validActions[rule.Action] {
			sendErrorResponse(w, fmt.Sprintf("Invalid action '%s' in rule %d. Must be one of: block, throttle, notify, log", rule.Action, i+1), http.StatusBadRequest)
			return
		}

		// For non-free rules, limit value is required
		if rule.RuleType != "free" && rule.LimitValue <= 0 {
			sendErrorResponse(w, fmt.Sprintf("Rule %d must have a positive limit_value", i+1), http.StatusBadRequest)
			return
		}

		// For time-based rules, period is required
		needsPeriod := rule.RuleType == "rate" || rule.RuleType == "token" || rule.RuleType == "time" || rule.RuleType == "credit"
		if needsPeriod && rule.Period == "" {
			sendErrorResponse(w, fmt.Sprintf("Rule %d requires a period", i+1), http.StatusBadRequest)
			return
		}

		// Validate period if provided
		if rule.Period != "" {
			validPeriods := map[string]bool{
				"minute": true,
				"hour":   true,
				"day":    true,
				"week":   true,
				"month":  true,
				"year":   true,
			}

			if !validPeriods[rule.Period] {
				sendErrorResponse(w, fmt.Sprintf("Invalid period '%s' in rule %d. Must be one of: minute, hour, day, week, month, year", rule.Period, i+1), http.StatusBadRequest)
				return
			}
		}
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Start transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Will be a no-op if transaction succeeds

	// Create policy
	policy := &db.Policy{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   currentUserID,
	}

	if err := db.CreatePolicyTx(tx, policy); err != nil {
		sendErrorResponse(w, "Failed to create policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create rules
	for _, ruleReq := range req.Rules {
		rule := &db.PolicyRule{
			ID:         uuid.New().String(),
			PolicyID:   policy.ID,
			RuleType:   ruleReq.RuleType,
			LimitValue: ruleReq.LimitValue,
			Period:     ruleReq.Period,
			Action:     ruleReq.Action,
			CreatedAt:  time.Now(),
		}

		// Set priority if provided, otherwise use default
		if ruleReq.Priority > 0 {
			rule.Priority = ruleReq.Priority
		} else {
			rule.Priority = 100 // Default priority
		}

		if err := db.CreatePolicyRuleTx(tx, rule); err != nil {
			sendErrorResponse(w, "Failed to create policy rule: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get full policy with rules for response
	createdPolicy, err := db.GetPolicyWithRules(database, policy.ID)
	if err != nil {
		// This shouldn't prevent returning success, but log it
		utils.LogError(ctx, "Failed to get created policy with rules: %v", err)

		// Send a simplified response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(policy)
		return
	}

	// Convert to response format
	ruleDetails := make([]PolicyRuleDetail, 0, len(createdPolicy.Rules))
	for _, rule := range createdPolicy.Rules {
		ruleDetail := PolicyRuleDetail{
			Type:   rule.RuleType,
			Limit:  rule.LimitValue,
			Period: rule.Period,
			Action: rule.Action,
		}
		ruleDetails = append(ruleDetails, ruleDetail)
	}

	response := PolicyDetail{
		ID:    createdPolicy.ID,
		Name:  createdPolicy.Name,
		Type:  createdPolicy.Type,
		Rules: ruleDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleUpdatePolicy handles PATCH /api/policies/:id
func HandleUpdatePolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get policy ID from path
	policyID := getPathParam(r, "id")
	if policyID == "" {
		sendErrorResponse(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get the current policy
	policy, err := db.GetPolicy(database, policyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Policy not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve policy: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Only the creator or local user can update policies
	if policy.CreatedBy != currentUserID && currentUserID != "local-user" {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Make sure we're updating at least one field
	if req.Name == nil && req.Description == nil && req.IsActive == nil && len(req.Rules) == 0 {
		sendErrorResponse(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Will be a no-op if transaction succeeds

	// Update policy fields
	if req.Name != nil {
		policy.Name = *req.Name
	}

	if req.Description != nil {
		policy.Description = *req.Description
	}

	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}

	policy.UpdatedAt = time.Now()

	// Update policy record
	if err := db.UpdatePolicyTx(tx, policy); err != nil {
		sendErrorResponse(w, "Failed to update policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update rules if provided
	if len(req.Rules) > 0 {
		// Delete existing rules
		if err := db.DeletePolicyRulesTx(tx, policyID); err != nil {
			sendErrorResponse(w, "Failed to delete existing rules: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Create new rules
		for _, ruleReq := range req.Rules {
			rule := &db.PolicyRule{
				ID:         uuid.New().String(),
				PolicyID:   policy.ID,
				RuleType:   ruleReq.RuleType,
				LimitValue: ruleReq.LimitValue,
				Period:     ruleReq.Period,
				Action:     ruleReq.Action,
				CreatedAt:  time.Now(),
			}

			// Set priority if provided, otherwise use default
			if ruleReq.Priority > 0 {
				rule.Priority = ruleReq.Priority
			} else {
				rule.Priority = 100 // Default priority
			}

			if err := db.CreatePolicyRuleTx(tx, rule); err != nil {
				sendErrorResponse(w, "Failed to create policy rule: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated policy with rules for response
	updatedPolicy, err := db.GetPolicyWithRules(database, policy.ID)
	if err != nil {
		// This shouldn't prevent returning success, but log it
		utils.LogError(ctx, "Failed to get updated policy with rules: %v", err)

		// Send a simplified response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(policy)
		return
	}

	// Convert to response format
	ruleDetails := make([]PolicyRuleDetail, 0, len(updatedPolicy.Rules))
	for _, rule := range updatedPolicy.Rules {
		ruleDetail := PolicyRuleDetail{
			Type:   rule.RuleType,
			Limit:  rule.LimitValue,
			Period: rule.Period,
			Action: rule.Action,
		}
		ruleDetails = append(ruleDetails, ruleDetail)
	}

	response := PolicyDetail{
		ID:    updatedPolicy.ID,
		Name:  updatedPolicy.Name,
		Type:  updatedPolicy.Type,
		Rules: ruleDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleChangeAPIPolicy handles POST /api/apis/:id/policy
func HandleChangeAPIPolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID from path
	apiID := getPathParam(r, "id")
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	var req ChangePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.PolicyID == "" {
		sendErrorResponse(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	if !req.EffectiveImmediately && req.ScheduledDate == nil {
		sendErrorResponse(w, "Either effective_immediately must be true or scheduled_date must be provided", http.StatusBadRequest)
		return
	}

	// If scheduled date is provided, ensure it's in the future
	if req.ScheduledDate != nil && req.ScheduledDate.Before(time.Now()) {
		sendErrorResponse(w, "Scheduled date must be in the future", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Verify the policy exists
	policy, err := db.GetPolicy(database, req.PolicyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Policy not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve policy: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Verify policy is active
	if !policy.IsActive {
		sendErrorResponse(w, "Cannot assign inactive policy", http.StatusBadRequest)
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Only the host user can change API policy
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Determine effective date
	var effectiveDate *time.Time
	if req.EffectiveImmediately {
		now := time.Now()
		effectiveDate = &now
	} else {
		effectiveDate = req.ScheduledDate
	}

	// Start transaction
	tx, err := database.Begin()
	if err != nil {
		sendErrorResponse(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Will be a no-op if transaction succeeds

	// Get current policy ID for history record
	var oldPolicyID *string
	if api.PolicyID != nil {
		oldPolicyID = api.PolicyID
	}

	// Create policy change record
	policyChange := &db.PolicyChange{
		ID:            uuid.New().String(),
		APIID:         apiID,
		OldPolicyID:   oldPolicyID,
		NewPolicyID:   &req.PolicyID,
		ChangedAt:     time.Now(),
		ChangedBy:     currentUserID,
		EffectiveDate: effectiveDate,
		ChangeReason:  req.ChangeReason,
	}

	if err := db.CreatePolicyChangeTx(tx, policyChange); err != nil {
		sendErrorResponse(w, "Failed to record policy change: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply the policy change immediately if requested
	if req.EffectiveImmediately {
		api.PolicyID = &req.PolicyID
		api.UpdatedAt = time.Now()

		if err := db.UpdateAPITx(tx, api); err != nil {
			sendErrorResponse(w, "Failed to update API: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create response
	var oldPolicy *PolicyRef
	if oldPolicyID != nil {
		oldPolicyObj, err := db.GetPolicy(database, *oldPolicyID)
		if err == nil {
			oldPolicy = &PolicyRef{
				ID:   oldPolicyObj.ID,
				Name: oldPolicyObj.Name,
				Type: oldPolicyObj.Type,
			}
		}
	}

	newPolicy := &PolicyRef{
		ID:   policy.ID,
		Name: policy.Name,
		Type: policy.Type,
	}

	response := PolicyChangeResponse{
		ID:            policyChange.ID,
		APIID:         policyChange.APIID,
		OldPolicy:     oldPolicy,
		NewPolicy:     newPolicy,
		ChangedAt:     policyChange.ChangedAt,
		ChangedBy:     policyChange.ChangedBy,
		EffectiveDate: policyChange.EffectiveDate,
		ChangeReason:  policyChange.ChangeReason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDeletePolicy handles DELETE /api/policies/:id
func HandleDeletePolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get Policy ID from path
	policyID := getPathParam(r, "id")
	if policyID == "" {
		sendErrorResponse(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// First check if policy is being used by any APIs
	_, total, err := db.ListAPIsByPolicy(database, policyID, 1, 0, "", "")
	if err != nil {
		sendErrorResponse(w, "Failed to check policy usage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If policy is in use, don't allow deletion
	if total > 0 {
		sendErrorResponse(w, fmt.Sprintf("Cannot delete policy because it is currently used by %d APIs", total), http.StatusBadRequest)
		return
	}

	// Delete the policy
	err = db.DeletePolicy(database, policyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "Policy not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to delete policy: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Delete all policy rules as well
	err = db.DeletePolicyRules(database, policyID)
	if err != nil {
		// Log the error but don't fail the request since the policy was deleted
		utils.LogError(ctx, "Failed to delete policy rules: %v", err)
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetAPIsByPolicy handles GET /api/policies/:id/apis
func HandleGetAPIsByPolicy(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get Policy ID from path
	policyID := getPathParam(r, "id")
	if policyID == "" {
		sendErrorResponse(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	// Parse sorting parameters
	sort := "created_at" // default
	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		if sortParam == "name" || sortParam == "created_at" {
			sort = sortParam
		}
	}

	order := "desc" // default
	if orderParam := r.URL.Query().Get("order"); orderParam != "" {
		if orderParam == "asc" || orderParam == "desc" {
			order = orderParam
		}
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Get all APIs that use this policy
	apis, total, err := db.ListAPIsByPolicy(database, policyID, limit, offset, sort, order)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve APIs by policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	apiList := make([]APIBasic, 0, len(apis))
	for _, api := range apis {
		// Get external user count
		userCount, err := db.CountAPIExternalUsers(database, api.ID)
		if err != nil {
			sendErrorResponse(w, "Failed to count external users: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Get document count
		docCount, err := db.CountAPIDocuments(database, api.ID)
		if err != nil {
			sendErrorResponse(w, "Failed to count documents: "+err.Error(), http.StatusInternalServerError)
			return
		}

		apiBasic := APIBasic{
			ID:                 api.ID,
			Name:               api.Name,
			Description:        api.Description,
			IsActive:           api.IsActive,
			IsDeprecated:       api.IsDeprecated,
			CreatedAt:          api.CreatedAt,
			UpdatedAt:          api.UpdatedAt,
			ExternalUsersCount: userCount,
			DocumentsCount:     docCount,
		}

		apiList = append(apiList, apiBasic)
	}

	response := APIListResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		APIs:   apiList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetAPIPolicyHistory handles GET /api/apis/:id/policy/history
func HandleGetAPIPolicyHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get API ID from path
	apiID := getPathParam(r, "id")
	if apiID == "" {
		sendErrorResponse(w, "API ID is required", http.StatusBadRequest)
		return
	}

	// Get database connection from context
	database, err := utils.DBFromContext(ctx)
	if err != nil {
		sendErrorResponse(w, "Failed to get database connection", http.StatusInternalServerError)
		return
	}

	// Verify the API exists
	api, err := db.GetAPI(database, apiID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendErrorResponse(w, "API not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, "Failed to retrieve API: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get the current user ID
	currentUserID, err := utils.UserIDFromContext(ctx)
	if err != nil {
		// For development/testing - in production, should return an error
		currentUserID = "local-user"
	}

	// Only the host user can view policy history
	if currentUserID != "local-user" && currentUserID != api.HostUserID {
		sendErrorResponse(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get policy change history
	changes, err := db.GetPolicyChangeHistory(database, apiID)
	if err != nil {
		sendErrorResponse(w, "Failed to retrieve policy change history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	changeResponses := make([]PolicyChangeResponse, 0, len(changes))
	for _, change := range changes {
		var oldPolicy, newPolicy *PolicyRef

		// Get old policy details if available
		if change.OldPolicyID != nil {
			oldPolicyObj, err := db.GetPolicy(database, *change.OldPolicyID)
			if err == nil {
				oldPolicy = &PolicyRef{
					ID:   oldPolicyObj.ID,
					Name: oldPolicyObj.Name,
					Type: oldPolicyObj.Type,
				}
			}
		}

		// Get new policy details if available
		if change.NewPolicyID != nil {
			newPolicyObj, err := db.GetPolicy(database, *change.NewPolicyID)
			if err == nil {
				newPolicy = &PolicyRef{
					ID:   newPolicyObj.ID,
					Name: newPolicyObj.Name,
					Type: newPolicyObj.Type,
				}
			}
		}

		changeResponse := PolicyChangeResponse{
			ID:            change.ID,
			APIID:         change.APIID,
			OldPolicy:     oldPolicy,
			NewPolicy:     newPolicy,
			ChangedAt:     change.ChangedAt,
			ChangedBy:     change.ChangedBy,
			EffectiveDate: change.EffectiveDate,
			ChangeReason:  change.ChangeReason,
		}

		changeResponses = append(changeResponses, changeResponse)
	}

	response := PolicyChangeHistoryResponse{
		APIID:   apiID,
		Changes: changeResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
