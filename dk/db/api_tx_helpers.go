package db

import (
	"database/sql"
	"fmt"
	"time"
)

// UpdateAPITx updates an existing API within a transaction
func UpdateAPITx(tx *sql.Tx, api *API) error {
	// Update timestamp if not already set
	if api.UpdatedAt.IsZero() {
		api.UpdatedAt = time.Now()
	}

	query := `
		UPDATE apis
		SET name = ?, description = ?, updated_at = ?, is_active = ?, 
			api_key = ?, host_user_id = ?, policy_id = ?, is_deprecated = ?, 
			deprecation_date = ?, deprecation_message = ?
		WHERE id = ?
	`

	result, err := tx.Exec(
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
		return fmt.Errorf("failed to update API in transaction: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
