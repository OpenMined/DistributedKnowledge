package db

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// ListAPIsByPolicy retrieves all APIs that use a specific policy
func ListAPIsByPolicy(db *sql.DB, policyID string, limit, offset int, sort, order string) ([]*API, int, error) {
	// Build the query to get APIs by policy ID
	query := `
		SELECT id, name, description, created_at, updated_at, is_active,
			api_key, host_user_id, policy_id, is_deprecated,
			deprecation_date, deprecation_message
		FROM apis
		WHERE policy_id = ?
	`
	countQuery := "SELECT COUNT(*) FROM apis WHERE policy_id = ?"

	args := []interface{}{policyID}

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
	err := db.QueryRow(countQuery, policyID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count APIs by policy: %v", err)
	}

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query APIs by policy: %v", err)
	}
	defer rows.Close()

	// Process results
	apis := []*API{}
	for rows.Next() {
		api := &API{}
		var policyIDNullable sql.NullString
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
			&policyIDNullable,
			&api.IsDeprecated,
			&deprecationDate,
			&deprecationMessage,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API row: %v", err)
		}

		if policyIDNullable.Valid {
			policyIDStr := policyIDNullable.String
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

// CreatePolicy creates a new policy
func CreatePolicy(db *sql.DB, policy *Policy) error {
	// Generate UUID if not provided
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}

	// Set timestamps if not provided
	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = now
	}

	query := `
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Type,
		policy.IsActive,
		policy.CreatedAt,
		policy.UpdatedAt,
		policy.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %v", err)
	}

	return nil
}

// CreatePolicyTx creates a new policy within a transaction
func CreatePolicyTx(tx *sql.Tx, policy *Policy) error {
	// Generate UUID if not provided
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}

	// Set timestamps if not provided
	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = now
	}

	query := `
		INSERT INTO policies (id, name, description, type, is_active, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Type,
		policy.IsActive,
		policy.CreatedAt,
		policy.UpdatedAt,
		policy.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy in transaction: %v", err)
	}

	return nil
}
