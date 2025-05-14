package db

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// CreateQuotaNotification creates a new quota notification
func CreateQuotaNotification(db *sql.DB, notification *QuotaNotification) error {
	// Generate UUID if not provided
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO quota_notifications (
			id, api_id, external_user_id, notification_type, rule_type,
			percentage_used, message, created_at, is_read, read_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		notification.ID,
		notification.APIID,
		notification.ExternalUserID,
		notification.NotificationType,
		notification.RuleType,
		notification.PercentageUsed,
		notification.Message,
		notification.CreatedAt,
		notification.IsRead,
		notification.ReadAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create quota notification: %v", err)
	}

	return nil
}

// GetQuotaNotification retrieves a notification by ID
func GetQuotaNotification(db *sql.DB, id string) (*QuotaNotification, error) {
	query := `
		SELECT id, api_id, external_user_id, notification_type, rule_type,
			percentage_used, message, created_at, is_read, read_at
		FROM quota_notifications
		WHERE id = ?
	`

	notification := &QuotaNotification{}
	var ruleType sql.NullString
	var percentageUsed sql.NullFloat64
	var message sql.NullString
	var readAt sql.NullTime

	err := db.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.APIID,
		&notification.ExternalUserID,
		&notification.NotificationType,
		&ruleType,
		&percentageUsed,
		&message,
		&notification.CreatedAt,
		&notification.IsRead,
		&readAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get quota notification: %v", err)
	}

	if ruleType.Valid {
		notification.RuleType = ruleType.String
	}

	if percentageUsed.Valid {
		notification.PercentageUsed = percentageUsed.Float64
	}

	if message.Valid {
		notification.Message = message.String
	}

	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}

	return notification, nil
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(db *sql.DB, id string) error {
	query := `
		UPDATE quota_notifications
		SET is_read = TRUE, read_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %v", err)
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

// GetUserNotifications retrieves all notifications for a user
func GetUserNotifications(db *sql.DB, externalUserID string, unreadOnly bool, limit, offset int) ([]*QuotaNotification, int, error) {
	baseQuery := `FROM quota_notifications WHERE external_user_id = ?`
	args := []interface{}{externalUserID}

	if unreadOnly {
		baseQuery += " AND is_read = FALSE"
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %v", err)
	}

	// Main query with pagination
	query := `
		SELECT id, api_id, external_user_id, notification_type, rule_type,
			percentage_used, message, created_at, is_read, read_at
		` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %v", err)
	}
	defer rows.Close()

	notifications := []*QuotaNotification{}
	for rows.Next() {
		notification := &QuotaNotification{}
		var ruleType sql.NullString
		var percentageUsed sql.NullFloat64
		var message sql.NullString
		var readAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.APIID,
			&notification.ExternalUserID,
			&notification.NotificationType,
			&ruleType,
			&percentageUsed,
			&message,
			&notification.CreatedAt,
			&notification.IsRead,
			&readAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification row: %v", err)
		}

		if ruleType.Valid {
			notification.RuleType = ruleType.String
		}

		if percentageUsed.Valid {
			notification.PercentageUsed = percentageUsed.Float64
		}

		if message.Valid {
			notification.Message = message.String
		}

		if readAt.Valid {
			notification.ReadAt = &readAt.Time
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating notification rows: %v", err)
	}

	return notifications, total, nil
}

// GetAPINotifications retrieves all notifications for an API
func GetAPINotifications(db *sql.DB, apiID string, limit, offset int) ([]*QuotaNotification, int, error) {
	baseQuery := `FROM quota_notifications WHERE api_id = ?`
	args := []interface{}{apiID}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %v", err)
	}

	// Main query with pagination
	query := `
		SELECT id, api_id, external_user_id, notification_type, rule_type,
			percentage_used, message, created_at, is_read, read_at
		` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %v", err)
	}
	defer rows.Close()

	notifications := []*QuotaNotification{}
	for rows.Next() {
		notification := &QuotaNotification{}
		var ruleType sql.NullString
		var percentageUsed sql.NullFloat64
		var message sql.NullString
		var readAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.APIID,
			&notification.ExternalUserID,
			&notification.NotificationType,
			&ruleType,
			&percentageUsed,
			&message,
			&notification.CreatedAt,
			&notification.IsRead,
			&readAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification row: %v", err)
		}

		if ruleType.Valid {
			notification.RuleType = ruleType.String
		}

		if percentageUsed.Valid {
			notification.PercentageUsed = percentageUsed.Float64
		}

		if message.Valid {
			notification.Message = message.String
		}

		if readAt.Valid {
			notification.ReadAt = &readAt.Time
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating notification rows: %v", err)
	}

	return notifications, total, nil
}

// DeleteQuotaNotification deletes a notification
func DeleteQuotaNotification(db *sql.DB, id string) error {
	query := "DELETE FROM quota_notifications WHERE id = ?"

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %v", err)
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

// DeleteAllReadNotifications deletes all read notifications older than the specified age
func DeleteAllReadNotifications(db *sql.DB, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)
	query := "DELETE FROM quota_notifications WHERE is_read = TRUE AND created_at < ?"

	result, err := db.Exec(query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete read notifications: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %v", err)
	}

	return rowsAffected, nil
}
