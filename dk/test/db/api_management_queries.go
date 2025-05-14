package db

import (
	"database/sql"
	//"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// This file contains the CRUD operations for the API Management tables

// If ErrNotFound is already defined elsewhere, we can omit this definition
// var (
//     ErrNotFound = errors.New("not found")
// )

// CreateAPI inserts a new API record
func CreateAPI(db *sql.DB, api *API) error {
	// Generate UUID if not provided
	if api.ID == "" {
		api.ID = uuid.New().String()
	}

	// Generate API key if not provided
	if api.APIKey == "" {
		apiKey, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %v", err)
		}
		api.APIKey = apiKey
	}

	// Set timestamps
	now := time.Now()
	api.CreatedAt = now
	api.UpdatedAt = now

	query := `
		INSERT INTO apis (
			id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		api.ID,
		api.Name,
		api.Description,
		api.CreatedAt,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
	)

	return err
}

// CreateAPITx inserts a new API record within a transaction
func CreateAPITx(tx *sql.Tx, api *API) error {
	// Generate UUID if not provided
	if api.ID == "" {
		api.ID = uuid.New().String()
	}

	// Generate API key if not provided
	if api.APIKey == "" {
		apiKey, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %v", err)
		}
		api.APIKey = apiKey
	}

	// Set timestamps
	now := time.Now()
	api.CreatedAt = now
	api.UpdatedAt = now

	query := `
		INSERT INTO apis (
			id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		api.ID,
		api.Name,
		api.Description,
		api.CreatedAt,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
	)

	return err
}

// GetAPI retrieves an API by ID
func GetAPI(db *sql.DB, id string) (*API, error) {
	query := `
		SELECT id, name, description, created_at, updated_at, is_active, 
			api_key, host_user_id, policy_id, is_deprecated, 
			deprecation_date, deprecation_message
		FROM apis
		WHERE id = ?
	`

	api := &API{}
	var policyID sql.NullString
	var deprecationDate sql.NullTime
	var deprecationMessage sql.NullString

	err := db.QueryRow(query, id).Scan(
		&api.ID,
		&api.Name,
		&api.Description,
		&api.CreatedAt,
		&api.UpdatedAt,
		&api.IsActive,
		&api.APIKey,
		&api.HostUserID,
		&policyID,
		&api.IsDeprecated,
		&deprecationDate,
		&deprecationMessage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if policyID.Valid {
		api.PolicyID = &policyID.String
	}

	if deprecationDate.Valid {
		api.DeprecationDate = &deprecationDate.Time
	}

	if deprecationMessage.Valid {
		api.DeprecationMessage = deprecationMessage.String
	}

	return api, nil
}

// UpdateAPI updates an existing API record
func UpdateAPI(db *sql.DB, api *API) error {
	// Update timestamp
	api.UpdatedAt = time.Now()

	query := `
		UPDATE apis
		SET name = ?, description = ?, updated_at = ?, is_active = ?, 
			api_key = ?, host_user_id = ?, policy_id = ?, is_deprecated = ?, 
			deprecation_date = ?, deprecation_message = ?
		WHERE id = ?
	`

	result, err := db.Exec(
		query,
		api.Name,
		api.Description,
		api.UpdatedAt,
		api.IsActive,
		api.APIKey,
		api.HostUserID,
		api.PolicyID,
		api.IsDeprecated,
		api.DeprecationDate,
		api.DeprecationMessage,
		api.ID,
	)

	if err != nil {
		return err
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAPI deletes an API record
func DeleteAPI(db *sql.DB, id string) error {
	query := "DELETE FROM apis WHERE id = ?"
	result, err := db.Exec(query, id)
	if err != nil {
		return err
	}

	// Check if any row was affected
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ListAPIs retrieves a paginated, filtered list of APIs
func ListAPIs(db *sql.DB, status, externalUserID string, limit, offset int, sort, order string) ([]*API, int, error) {
	// Build the query based on filters
	query := "SELECT id, name, description, created_at, updated_at, is_active, api_key, host_user_id, policy_id, is_deprecated, deprecation_date, deprecation_message FROM apis WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM apis WHERE 1=1"

	args := []interface{}{}

	// Apply status filter
	if status != "" {
		switch status {
		case "active":
			query += " AND is_active = TRUE AND is_deprecated = FALSE"
			countQuery += " AND is_active = TRUE AND is_deprecated = FALSE"
		case "inactive":
			query += " AND is_active = FALSE AND is_deprecated = FALSE"
			countQuery += " AND is_active = FALSE AND is_deprecated = FALSE"
		case "deprecated":
			query += " AND is_deprecated = TRUE"
			countQuery += " AND is_deprecated = TRUE"
		}
	}

	// Apply external user filter
	if externalUserID != "" {
		query += " AND id IN (SELECT api_id FROM api_user_access WHERE external_user_id = ? AND is_active = TRUE)"
		countQuery += " AND id IN (SELECT api_id FROM api_user_access WHERE external_user_id = ? AND is_active = TRUE)"
		args = append(args, externalUserID)
	}

	// Apply sorting
	if sort == "" {
		sort = "created_at" // default
	}

	if sort != "name" && sort != "created_at" {
		sort = "created_at" // fallback to default for invalid sort fields
	}

	query += " ORDER BY " + sort
	if order != "" {
		if order != "asc" && order != "desc" {
			order = "desc" // fallback to default for invalid order
		}
		query += " " + order
	} else {
		query += " DESC" // default order
	}

	// Apply pagination
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count APIs: %v", err)
	}

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query APIs: %v", err)
	}
	defer rows.Close()

	// Process results
	apis := []*API{}
	for rows.Next() {
		api := &API{}
		var policyID sql.NullString
		var deprecationDate sql.NullTime
		var deprecationMessage sql.NullString

		err := rows.Scan(
			&api.ID,
			&api.Name,
			&api.Description,
			&api.CreatedAt,
			&api.UpdatedAt,
			&api.IsActive,
			&api.APIKey,
			&api.HostUserID,
			&policyID,
			&api.IsDeprecated,
			&deprecationDate,
			&deprecationMessage,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API row: %v", err)
		}

		if policyID.Valid {
			policyIDStr := policyID.String
			api.PolicyID = &policyIDStr
		}

		if deprecationDate.Valid {
			api.DeprecationDate = &deprecationDate.Time
		}

		if deprecationMessage.Valid {
			api.DeprecationMessage = deprecationMessage.String
		}

		apis = append(apis, api)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating API rows: %v", err)
	}

	return apis, total, nil
}

// CountAPIExternalUsers counts how many external users have access to an API
func CountAPIExternalUsers(db *sql.DB, apiID string) (int, error) {
	query := "SELECT COUNT(*) FROM api_user_access WHERE api_id = ? AND is_active = TRUE"
	var count int
	err := db.QueryRow(query, apiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API external users: %v", err)
	}
	return count, nil
}

// CountAPIDocuments counts how many documents are associated with an API
func CountAPIDocuments(db *sql.DB, apiID string) (int, error) {
	query := "SELECT COUNT(*) FROM document_associations WHERE entity_id = ? AND entity_type = 'api'"
	var count int
	err := db.QueryRow(query, apiID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API documents: %v", err)
	}
	return count, nil
}

// GetAPIExternalUsers retrieves all external users with access to an API
func GetAPIExternalUsers(db *sql.DB, apiID string) ([]*APIUserAccess, error) {
	query := `
		SELECT id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active
		FROM api_user_access
		WHERE api_id = ? AND is_active = TRUE
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API external users: %v", err)
	}
	defer rows.Close()

	users := []*APIUserAccess{}
	for rows.Next() {
		user := &APIUserAccess{}
		var revokedAt sql.NullTime
		var grantedBy sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.APIID,
			&user.ExternalUserID,
			&user.AccessLevel,
			&user.GrantedAt,
			&grantedBy,
			&revokedAt,
			&user.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API user access row: %v", err)
		}

		if grantedBy.Valid {
			user.GrantedBy = grantedBy.String
		}

		if revokedAt.Valid {
			user.RevokedAt = &revokedAt.Time
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API user access rows: %v", err)
	}

	return users, nil
}

// GetAPIDocuments retrieves all documents associated with an API
func GetAPIDocuments(db *sql.DB, apiID string) ([]*DocumentAssociation, error) {
	query := `
		SELECT id, document_filename, entity_id, entity_type, created_at
		FROM document_associations
		WHERE entity_id = ? AND entity_type = 'api'
	`

	rows, err := db.Query(query, apiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API documents: %v", err)
	}
	defer rows.Close()

	docs := []*DocumentAssociation{}
	for rows.Next() {
		doc := &DocumentAssociation{}

		err := rows.Scan(
			&doc.ID,
			&doc.DocumentFilename,
			&doc.EntityID,
			&doc.EntityType,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document association row: %v", err)
		}

		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating document association rows: %v", err)
	}

	return docs, nil
}

// CreateDocumentAssociation inserts a new document association
func CreateDocumentAssociation(db *sql.DB, assoc *DocumentAssociation) error {
	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	// Set timestamp
	if assoc.CreatedAt.IsZero() {
		assoc.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO document_associations (id, document_filename, entity_id, entity_type, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		assoc.ID,
		assoc.DocumentFilename,
		assoc.EntityID,
		assoc.EntityType,
		assoc.CreatedAt,
	)

	return err
}

// CreateDocumentAssociationTx inserts a new document association within a transaction
func CreateDocumentAssociationTx(tx *sql.Tx, assoc *DocumentAssociation) error {
	// Generate UUID if not provided
	if assoc.ID == "" {
		assoc.ID = uuid.New().String()
	}

	// Set timestamp
	if assoc.CreatedAt.IsZero() {
		assoc.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO document_associations (id, document_filename, entity_id, entity_type, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		assoc.ID,
		assoc.DocumentFilename,
		assoc.EntityID,
		assoc.EntityType,
		assoc.CreatedAt,
	)

	return err
}

// CreateAPIUserAccess inserts a new API user access record
func CreateAPIUserAccess(db *sql.DB, access *APIUserAccess) error {
	// Generate UUID if not provided
	if access.ID == "" {
		access.ID = uuid.New().String()
	}

	// Set timestamp
	if access.GrantedAt.IsZero() {
		access.GrantedAt = time.Now()
	}

	query := `
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		access.ID,
		access.APIID,
		access.ExternalUserID,
		access.AccessLevel,
		access.GrantedAt,
		access.GrantedBy,
		access.RevokedAt,
		access.IsActive,
	)

	return err
}

// CreateAPIUserAccessTx inserts a new API user access record within a transaction
func CreateAPIUserAccessTx(tx *sql.Tx, access *APIUserAccess) error {
	// Generate UUID if not provided
	if access.ID == "" {
		access.ID = uuid.New().String()
	}

	// Set timestamp
	if access.GrantedAt.IsZero() {
		access.GrantedAt = time.Now()
	}

	query := `
		INSERT INTO api_user_access (id, api_id, external_user_id, access_level, granted_at, granted_by, revoked_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		access.ID,
		access.APIID,
		access.ExternalUserID,
		access.AccessLevel,
		access.GrantedAt,
		access.GrantedBy,
		access.RevokedAt,
		access.IsActive,
	)

	return err
}

// GetPolicy retrieves a policy by ID
func GetPolicy(db *sql.DB, id string) (*Policy, error) {
	query := `
		SELECT id, name, description, type, is_active, created_at, updated_at, created_by
		FROM policies
		WHERE id = ?
	`

	policy := &Policy{}
	var createdBy sql.NullString
	var description sql.NullString

	err := db.QueryRow(query, id).Scan(
		&policy.ID,
		&policy.Name,
		&description,
		&policy.Type,
		&policy.IsActive,
		&policy.CreatedAt,
		&policy.UpdatedAt,
		&createdBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if description.Valid {
		policy.Description = description.String
	}

	if createdBy.Valid {
		policy.CreatedBy = createdBy.String
	}

	return policy, nil
}

// GetPolicyWithRules retrieves a policy with its rules by ID
func GetPolicyWithRules(db *sql.DB, id string) (*Policy, error) {
	// Get the policy first
	policy, err := GetPolicy(db, id)
	if err != nil {
		return nil, err
	}

	// Then get the rules
	query := `
		SELECT id, policy_id, rule_type, limit_value, period, action, priority, created_at
		FROM policy_rules
		WHERE policy_id = ?
		ORDER BY priority
	`

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy rules: %v", err)
	}
	defer rows.Close()

	rules := []PolicyRule{}
	for rows.Next() {
		rule := PolicyRule{}
		var period sql.NullString
		var limitValue sql.NullFloat64

		err := rows.Scan(
			&rule.ID,
			&rule.PolicyID,
			&rule.RuleType,
			&limitValue,
			&period,
			&rule.Action,
			&rule.Priority,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy rule row: %v", err)
		}

		if period.Valid {
			rule.Period = period.String
		}

		if limitValue.Valid {
			rule.LimitValue = limitValue.Float64
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy rule rows: %v", err)
	}

	policy.Rules = rules

	return policy, nil
}

// CreatePolicyChange records a policy change
func CreatePolicyChange(db *sql.DB, change *PolicyChange) error {
	// Generate UUID if not provided
	if change.ID == "" {
		change.ID = uuid.New().String()
	}

	// Set timestamp
	if change.ChangedAt.IsZero() {
		change.ChangedAt = time.Now()
	}

	query := `
		INSERT INTO policy_changes (id, api_id, old_policy_id, new_policy_id, changed_at, changed_by, effective_date, change_reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		change.ID,
		change.APIID,
		change.OldPolicyID,
		change.NewPolicyID,
		change.ChangedAt,
		change.ChangedBy,
		change.EffectiveDate,
		change.ChangeReason,
	)

	return err
}

// GetAPIUsageSummaryByPeriod retrieves usage summary for an API by period
func GetAPIUsageSummaryByPeriod(db *sql.DB, apiID, periodType, periodValue string) (*APIUsageSummary, error) {
	query := `
		SELECT id, api_id, external_user_id, period_type, period_start, period_end, 
			total_requests, total_tokens, total_credits, total_time_ms, 
			throttled_requests, blocked_requests, last_updated
		FROM api_usage_summary
		WHERE api_id = ? AND period_type = ? AND period_start = ?
	`

	var periodStart time.Time
	var err error

	// Parse period value based on period type
	switch periodType {
	case "daily":
		periodStart, err = time.Parse("2006-01-02", periodValue)
	case "monthly":
		periodStart, err = time.Parse("2006-01", periodValue)
		// Add logic for other period types as needed
	}

	if err != nil {
		return nil, fmt.Errorf("invalid period value for %s: %v", periodType, err)
	}

	summary := &APIUsageSummary{}
	var externalUserID sql.NullString

	err = db.QueryRow(query, apiID, periodType, periodStart).Scan(
		&summary.ID,
		&summary.APIID,
		&externalUserID,
		&summary.PeriodType,
		&summary.PeriodStart,
		&summary.PeriodEnd,
		&summary.TotalRequests,
		&summary.TotalTokens,
		&summary.TotalCredits,
		&summary.TotalTimeMs,
		&summary.ThrottledRequests,
		&summary.BlockedRequests,
		&summary.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle nullable fields
	if externalUserID.Valid {
		summary.ExternalUserID = externalUserID.String
	}

	return summary, nil
}

// CreateAPIRequest inserts a new API request record
func CreateAPIRequest(db *sql.DB, request *APIRequest) error {
	// Generate UUID if not provided
	if request.ID == "" {
		request.ID = uuid.New().String()
	}

	// Set timestamps
	if request.SubmittedDate.IsZero() {
		request.SubmittedDate = time.Now()
	}

	query := `
		INSERT INTO api_requests (
			id, api_name, description, submitted_date, status, 
			requester_id, denial_reason, denied_date, approved_date, 
			submission_count, previous_request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		request.ID,
		request.APIName,
		request.Description,
		request.SubmittedDate,
		request.Status,
		request.RequesterID,
		request.DenialReason,
		request.DeniedDate,
		request.ApprovedDate,
		request.SubmissionCount,
		request.PreviousRequestID,
	)

	return err
}

// Helper function to generate a secure API key
func generateAPIKey() (string, error) {
	// Example implementation using UUID as the base
	key := fmt.Sprintf("api_%s", uuid.New().String())
	return key, nil
}
