package db

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// RecordAPIUsage records a single API usage entry
func RecordAPIUsage(db *sql.DB, usage *APIUsage) error {
	// Generate UUID if not provided
	if usage.ID == "" {
		usage.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	query := `
		INSERT INTO api_usage (
			id, api_id, external_user_id, timestamp, request_count,
			tokens_used, credits_consumed, execution_time_ms, endpoint,
			was_throttled, was_blocked
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		usage.ID,
		usage.APIID,
		usage.ExternalUserID,
		usage.Timestamp,
		usage.RequestCount,
		usage.TokensUsed,
		usage.CreditsConsumed,
		usage.ExecutionTimeMs,
		usage.Endpoint,
		usage.WasThrottled,
		usage.WasBlocked,
	)

	if err != nil {
		return fmt.Errorf("failed to record API usage: %v", err)
	}

	return nil
}

// RecordAPIUsageTx records a single API usage entry within a transaction
func RecordAPIUsageTx(tx *sql.Tx, usage *APIUsage) error {
	// Generate UUID if not provided
	if usage.ID == "" {
		usage.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	query := `
		INSERT INTO api_usage (
			id, api_id, external_user_id, timestamp, request_count,
			tokens_used, credits_consumed, execution_time_ms, endpoint,
			was_throttled, was_blocked
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		usage.ID,
		usage.APIID,
		usage.ExternalUserID,
		usage.Timestamp,
		usage.RequestCount,
		usage.TokensUsed,
		usage.CreditsConsumed,
		usage.ExecutionTimeMs,
		usage.Endpoint,
		usage.WasThrottled,
		usage.WasBlocked,
	)

	if err != nil {
		return fmt.Errorf("failed to record API usage in transaction: %v", err)
	}

	return nil
}

// GetRecentAPIUsage retrieves recent API usage records for an API and user
func GetRecentAPIUsage(db *sql.DB, apiID, externalUserID string, limit int) ([]*APIUsage, error) {
	query := `
		SELECT id, api_id, external_user_id, timestamp, request_count,
			tokens_used, credits_consumed, execution_time_ms, endpoint,
			was_throttled, was_blocked
		FROM api_usage
		WHERE api_id = ? AND external_user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, apiID, externalUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent API usage: %v", err)
	}
	defer rows.Close()

	usageRecords := []*APIUsage{}
	for rows.Next() {
		usage := &APIUsage{}
		var endpoint sql.NullString

		err := rows.Scan(
			&usage.ID,
			&usage.APIID,
			&usage.ExternalUserID,
			&usage.Timestamp,
			&usage.RequestCount,
			&usage.TokensUsed,
			&usage.CreditsConsumed,
			&usage.ExecutionTimeMs,
			&endpoint,
			&usage.WasThrottled,
			&usage.WasBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API usage row: %v", err)
		}

		if endpoint.Valid {
			usage.Endpoint = endpoint.String
		}

		usageRecords = append(usageRecords, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API usage rows: %v", err)
	}

	return usageRecords, nil
}

// GetUsageByPeriod retrieves usage for a given period
func GetUsageByPeriod(db *sql.DB, apiID, externalUserID string, periodStart, periodEnd time.Time) ([]*APIUsage, error) {
	query := `
		SELECT id, api_id, external_user_id, timestamp, request_count,
			tokens_used, credits_consumed, execution_time_ms, endpoint,
			was_throttled, was_blocked
		FROM api_usage
		WHERE api_id = ? AND external_user_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`

	rows, err := db.Query(query, apiID, externalUserID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query API usage by period: %v", err)
	}
	defer rows.Close()

	usageRecords := []*APIUsage{}
	for rows.Next() {
		usage := &APIUsage{}
		var endpoint sql.NullString

		err := rows.Scan(
			&usage.ID,
			&usage.APIID,
			&usage.ExternalUserID,
			&usage.Timestamp,
			&usage.RequestCount,
			&usage.TokensUsed,
			&usage.CreditsConsumed,
			&usage.ExecutionTimeMs,
			&endpoint,
			&usage.WasThrottled,
			&usage.WasBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API usage row: %v", err)
		}

		if endpoint.Valid {
			usage.Endpoint = endpoint.String
		}

		usageRecords = append(usageRecords, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API usage rows: %v", err)
	}

	return usageRecords, nil
}

// GetTotalUsageForPeriod calculates the sum of usage metrics for a period
func GetTotalUsageForPeriod(db *sql.DB, apiID, externalUserID string, periodStart, periodEnd time.Time) (*APIUsageSummary, error) {
	query := `
		SELECT 
			SUM(request_count) AS total_requests,
			SUM(tokens_used) AS total_tokens,
			SUM(credits_consumed) AS total_credits,
			SUM(execution_time_ms) AS total_time_ms,
			SUM(CASE WHEN was_throttled = TRUE THEN 1 ELSE 0 END) AS throttled_requests,
			SUM(CASE WHEN was_blocked = TRUE THEN 1 ELSE 0 END) AS blocked_requests
		FROM api_usage
		WHERE api_id = ? AND external_user_id = ? AND timestamp BETWEEN ? AND ?
	`

	summary := &APIUsageSummary{
		APIID:          apiID,
		ExternalUserID: externalUserID,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}

	var totalRequests, totalTokens, totalTimeMs, throttledRequests, blockedRequests sql.NullInt64
	var totalCredits sql.NullFloat64

	err := db.QueryRow(query, apiID, externalUserID, periodStart, periodEnd).Scan(
		&totalRequests,
		&totalTokens,
		&totalCredits,
		&totalTimeMs,
		&throttledRequests,
		&blockedRequests,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate total usage: %v", err)
	}

	// Handle nullable values
	if totalRequests.Valid {
		summary.TotalRequests = int(totalRequests.Int64)
	}
	if totalTokens.Valid {
		summary.TotalTokens = int(totalTokens.Int64)
	}
	if totalCredits.Valid {
		summary.TotalCredits = totalCredits.Float64
	}
	if totalTimeMs.Valid {
		summary.TotalTimeMs = int(totalTimeMs.Int64)
	}
	if throttledRequests.Valid {
		summary.ThrottledRequests = int(throttledRequests.Int64)
	}
	if blockedRequests.Valid {
		summary.BlockedRequests = int(blockedRequests.Int64)
	}

	return summary, nil
}

// UpsertAPIUsageSummary creates or updates a usage summary record
func UpsertAPIUsageSummary(db *sql.DB, summary *APIUsageSummary) error {
	// Generate UUID if not provided
	if summary.ID == "" {
		summary.ID = uuid.New().String()
	}

	// Set last updated timestamp
	summary.LastUpdated = time.Now()

	// First check if the summary exists with count only
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM api_usage_summary WHERE api_id = ? AND external_user_id = ? AND period_type = ? AND period_start = ?",
		summary.APIID, summary.ExternalUserID, summary.PeriodType, summary.PeriodStart,
	).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to check for existing summary count: %v", err)
	}

	// If summary exists, get its ID separately
	var existingID string
	if count > 0 {
		err := db.QueryRow(
			"SELECT id FROM api_usage_summary WHERE api_id = ? AND external_user_id = ? AND period_type = ? AND period_start = ?",
			summary.APIID, summary.ExternalUserID, summary.PeriodType, summary.PeriodStart,
		).Scan(&existingID)

		if err != nil {
			return fmt.Errorf("failed to get existing summary ID: %v", err)
		}
	}

	if count > 0 {
		// Update existing summary
		summary.ID = existingID
		query := `
			UPDATE api_usage_summary
			SET period_end = ?,
				total_requests = ?,
				total_tokens = ?,
				total_credits = ?,
				total_time_ms = ?,
				throttled_requests = ?,
				blocked_requests = ?,
				last_updated = ?
			WHERE id = ?
		`

		_, err = db.Exec(
			query,
			summary.PeriodEnd,
			summary.TotalRequests,
			summary.TotalTokens,
			summary.TotalCredits,
			summary.TotalTimeMs,
			summary.ThrottledRequests,
			summary.BlockedRequests,
			summary.LastUpdated,
			summary.ID,
		)
	} else {
		// Insert new summary
		query := `
			INSERT INTO api_usage_summary (
				id, api_id, external_user_id, period_type, period_start, period_end,
				total_requests, total_tokens, total_credits, total_time_ms,
				throttled_requests, blocked_requests, last_updated
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = db.Exec(
			query,
			summary.ID,
			summary.APIID,
			summary.ExternalUserID,
			summary.PeriodType,
			summary.PeriodStart,
			summary.PeriodEnd,
			summary.TotalRequests,
			summary.TotalTokens,
			summary.TotalCredits,
			summary.TotalTimeMs,
			summary.ThrottledRequests,
			summary.BlockedRequests,
			summary.LastUpdated,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to upsert API usage summary: %v", err)
	}

	return nil
}

// GetAPIUsageSummaries retrieves usage summaries for an API with optional filtering
func GetAPIUsageSummaries(db *sql.DB, apiID, externalUserID, periodType string, fromDate, toDate time.Time) ([]*APIUsageSummary, error) {
	query := `
		SELECT id, api_id, external_user_id, period_type, period_start, period_end,
			total_requests, total_tokens, total_credits, total_time_ms,
			throttled_requests, blocked_requests, last_updated
		FROM api_usage_summary
		WHERE api_id = ?
	`
	args := []interface{}{apiID}

	if externalUserID != "" {
		query += " AND external_user_id = ?"
		args = append(args, externalUserID)
	}

	if periodType != "" {
		query += " AND period_type = ?"
		args = append(args, periodType)
	}

	if !fromDate.IsZero() {
		query += " AND period_start >= ?"
		args = append(args, fromDate)
	}

	if !toDate.IsZero() {
		query += " AND period_end <= ?"
		args = append(args, toDate)
	}

	query += " ORDER BY period_start DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query API usage summaries: %v", err)
	}
	defer rows.Close()

	summaries := []*APIUsageSummary{}
	for rows.Next() {
		summary := &APIUsageSummary{}

		err := rows.Scan(
			&summary.ID,
			&summary.APIID,
			&summary.ExternalUserID,
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
			return nil, fmt.Errorf("failed to scan API usage summary row: %v", err)
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API usage summary rows: %v", err)
	}

	return summaries, nil
}

// GetAllAPIUsage gets API usage for all APIs and users with optional date filtering
func GetAllAPIUsage(db *sql.DB, fromDate, toDate time.Time, limit, offset int) ([]*APIUsage, int, error) {
	baseQuery := `FROM api_usage WHERE 1=1`
	args := []interface{}{}

	if !fromDate.IsZero() {
		baseQuery += " AND timestamp >= ?"
		args = append(args, fromDate)
	}

	if !toDate.IsZero() {
		baseQuery += " AND timestamp <= ?"
		args = append(args, toDate)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API usage records: %v", err)
	}

	// Main query with pagination
	query := `
		SELECT id, api_id, external_user_id, timestamp, request_count,
			tokens_used, credits_consumed, execution_time_ms, endpoint,
			was_throttled, was_blocked
		` + baseQuery + `
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API usage: %v", err)
	}
	defer rows.Close()

	usageRecords := []*APIUsage{}
	for rows.Next() {
		usage := &APIUsage{}
		var endpoint sql.NullString

		err := rows.Scan(
			&usage.ID,
			&usage.APIID,
			&usage.ExternalUserID,
			&usage.Timestamp,
			&usage.RequestCount,
			&usage.TokensUsed,
			&usage.CreditsConsumed,
			&usage.ExecutionTimeMs,
			&endpoint,
			&usage.WasThrottled,
			&usage.WasBlocked,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan API usage row: %v", err)
		}

		if endpoint.Valid {
			usage.Endpoint = endpoint.String
		}

		usageRecords = append(usageRecords, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating API usage rows: %v", err)
	}

	return usageRecords, total, nil
}

// UpdateAPIUsageSummaries refreshes all usage summaries by recalculating from raw usage data
func UpdateAPIUsageSummaries(db *sql.DB) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Get all APIs
	rows, err := tx.Query("SELECT id FROM apis")
	if err != nil {
		return fmt.Errorf("failed to query APIs: %v", err)
	}
	defer rows.Close()

	var apiIDs []string
	for rows.Next() {
		var apiID string
		if err := rows.Scan(&apiID); err != nil {
			return fmt.Errorf("failed to scan API ID: %v", err)
		}
		apiIDs = append(apiIDs, apiID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating API rows: %v", err)
	}

	now := time.Now()

	// For each API, recalculate summaries
	for _, apiID := range apiIDs {
		// Daily summary (today)
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

		if err := updateSummaryForPeriod(tx, apiID, "daily", startOfDay, endOfDay); err != nil {
			return fmt.Errorf("failed to update daily summary: %v", err)
		}

		// Weekly summary (this week)
		daysSinceMonday := int(now.Weekday())
		if daysSinceMonday == 0 { // Sunday
			daysSinceMonday = 6
		} else {
			daysSinceMonday--
		}
		startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
		endOfWeek := startOfWeek.Add(7 * 24 * time.Hour).Add(-time.Second)

		if err := updateSummaryForPeriod(tx, apiID, "weekly", startOfWeek, endOfWeek); err != nil {
			return fmt.Errorf("failed to update weekly summary: %v", err)
		}

		// Monthly summary (this month)
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)

		if err := updateSummaryForPeriod(tx, apiID, "monthly", startOfMonth, endOfMonth); err != nil {
			return fmt.Errorf("failed to update monthly summary: %v", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// updateSummaryForPeriod updates summaries for a specific period
func updateSummaryForPeriod(tx *sql.Tx, apiID, periodType string, periodStart, periodEnd time.Time) error {
	// Get all users who have used this API in this period
	userQuery := `
		SELECT DISTINCT external_user_id
		FROM api_usage
		WHERE api_id = ? AND timestamp BETWEEN ? AND ?
	`

	userRows, err := tx.Query(userQuery, apiID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to query users: %v", err)
	}
	defer userRows.Close()

	var userIDs []string
	for userRows.Next() {
		var userID string
		if err := userRows.Scan(&userID); err != nil {
			return fmt.Errorf("failed to scan user ID: %v", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := userRows.Err(); err != nil {
		return fmt.Errorf("error iterating user rows: %v", err)
	}

	// For each user, calculate and update summary
	for _, userID := range userIDs {
		// Calculate totals
		var totalRequests, totalTokens, totalTimeMs, throttledRequests, blockedRequests int64
		var totalCredits float64

		query := `
			SELECT 
				COALESCE(SUM(request_count), 0),
				COALESCE(SUM(tokens_used), 0),
				COALESCE(SUM(credits_consumed), 0),
				COALESCE(SUM(execution_time_ms), 0),
				COALESCE(SUM(CASE WHEN was_throttled = TRUE THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN was_blocked = TRUE THEN 1 ELSE 0 END), 0)
			FROM api_usage
			WHERE api_id = ? AND external_user_id = ? AND timestamp BETWEEN ? AND ?
		`

		err := tx.QueryRow(query, apiID, userID, periodStart, periodEnd).Scan(
			&totalRequests,
			&totalTokens,
			&totalCredits,
			&totalTimeMs,
			&throttledRequests,
			&blockedRequests,
		)

		if err != nil {
			return fmt.Errorf("failed to calculate usage totals: %v", err)
		}

		// Check if summary already exists - first get count
		var count int
		err = tx.QueryRow(
			"SELECT COUNT(*) FROM api_usage_summary WHERE api_id = ? AND external_user_id = ? AND period_type = ? AND period_start = ?",
			apiID, userID, periodType, periodStart,
		).Scan(&count)

		if err != nil {
			return fmt.Errorf("failed to check for existing summary count: %v", err)
		}

		// If a summary exists, get its ID
		var existingID string
		if count > 0 {
			err = tx.QueryRow(
				"SELECT id FROM api_usage_summary WHERE api_id = ? AND external_user_id = ? AND period_type = ? AND period_start = ?",
				apiID, userID, periodType, periodStart,
			).Scan(&existingID)

			if err != nil {
				return fmt.Errorf("failed to get existing summary ID: %v", err)
			}
		}

		now := time.Now()

		if count > 0 {
			// Update existing summary
			query := `
				UPDATE api_usage_summary
				SET period_end = ?,
					total_requests = ?,
					total_tokens = ?,
					total_credits = ?,
					total_time_ms = ?,
					throttled_requests = ?,
					blocked_requests = ?,
					last_updated = ?
				WHERE id = ?
			`

			_, err = tx.Exec(
				query,
				periodEnd,
				totalRequests,
				totalTokens,
				totalCredits,
				totalTimeMs,
				throttledRequests,
				blockedRequests,
				now,
				existingID,
			)
		} else {
			// Insert new summary
			query := `
				INSERT INTO api_usage_summary (
					id, api_id, external_user_id, period_type, period_start, period_end,
					total_requests, total_tokens, total_credits, total_time_ms,
					throttled_requests, blocked_requests, last_updated
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`

			_, err = tx.Exec(
				query,
				uuid.New().String(),
				apiID,
				userID,
				periodType,
				periodStart,
				periodEnd,
				totalRequests,
				totalTokens,
				totalCredits,
				totalTimeMs,
				throttledRequests,
				blockedRequests,
				now,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to upsert API usage summary: %v", err)
		}
	}

	return nil
}
