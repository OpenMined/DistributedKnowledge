package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AppRequest mirrors the app_requests table schema.
type AppRequest struct {
	AppName        string    `json:"app_name"`
	RequestedBy    string    `json:"requested_by"`
	AppDescription string    `json:"description"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason"`
	Safety         string    `json:"safety"`
	CreatedAt      time.Time `json:"created_at"`
}

// InsertOrUpdateAppRequest upserts a single request.
func InsertOrUpdateAppRequest(ctx context.Context, db *sql.DB, ar AppRequest) error {
	_, err := db.ExecContext(ctx, `
        INSERT INTO app_requests
          (app_name, requested_by, app_description, status, reason, safety)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(app_name) DO UPDATE SET
          requested_by=excluded.requested_by,
          app_description=excluded.app_description,
          status=excluded.status,
          reason=excluded.reason,
          safety=excluded.safety
    `, ar.AppName, ar.RequestedBy, ar.AppDescription, ar.Status, ar.Reason, ar.Safety)
	if err != nil {
		return fmt.Errorf("app_requests upsert: %w", err)
	}
	return nil
}

// GetAppRequest fetches one by name. Returns sql.ErrNoRows if none.
func GetAppRequest(ctx context.Context, db *sql.DB, name string) (AppRequest, error) {
	var ar AppRequest
	err := db.QueryRowContext(ctx, `
        SELECT app_name, requested_by, app_description, status, reason, safety, created_at
          FROM app_requests
         WHERE app_name = ?
    `, name).Scan(
		&ar.AppName,
		&ar.RequestedBy,
		&ar.AppDescription,
		&ar.Status,
		&ar.Reason,
		&ar.Safety,
		&ar.CreatedAt,
	)
	if err != nil {
		return AppRequest{}, err
	}
	return ar, nil
}

// ListPendingAppRequests returns all rows where status = 'pending' (optional).
func ListPendingAppRequests(ctx context.Context, db *sql.DB) ([]AppRequest, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT app_name, requested_by, app_description, status, reason, safety, created_at
          FROM app_requests
         WHERE LOWER(status) = 'pending'
      ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("list pending app_requests: %w", err)
	}
	defer rows.Close()

	var out []AppRequest
	for rows.Next() {
		var ar AppRequest
		if err := rows.Scan(
			&ar.AppName,
			&ar.RequestedBy,
			&ar.AppDescription,
			&ar.Status,
			&ar.Reason,
			&ar.Safety,
			&ar.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan app_request: %w", err)
		}
		out = append(out, ar)
	}
	return out, rows.Err()
}
