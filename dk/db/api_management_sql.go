package db

import (
	"database/sql"
	"fmt"
)

// RunAPIMigrations creates the necessary tables for the API Management System if they do not exist.
func RunAPIMigrations(db *sql.DB) error {
	// APIs entities table
	apisTable := `
	CREATE TABLE IF NOT EXISTS apis (
		id TEXT PRIMARY KEY,                          -- UUID for API
		name TEXT NOT NULL,                          -- API name
		description TEXT,                             -- API description
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active BOOLEAN DEFAULT FALSE,
		api_key TEXT UNIQUE,
		host_user_id TEXT NOT NULL,                   -- Always the local user
		policy_id TEXT,                              -- Reference to the assigned policy
		is_deprecated BOOLEAN DEFAULT FALSE,
		deprecation_date DATETIME,
		deprecation_message TEXT,
		FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE SET NULL
	);`

	// API access requests from external users
	apiRequestsTable := `
	CREATE TABLE IF NOT EXISTS api_requests (
		id TEXT PRIMARY KEY,                          -- UUID for request
		api_name TEXT NOT NULL,
		description TEXT,
		submitted_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT CHECK (status IN ('pending', 'approved', 'denied')) DEFAULT 'pending',
		requester_id TEXT NOT NULL,                   -- External user requesting access
		denial_reason TEXT,
		denied_date DATETIME,
		approved_date DATETIME,
		submission_count INTEGER DEFAULT 1,
		previous_request_id TEXT,
		proposed_policy_id TEXT,
		FOREIGN KEY (previous_request_id) REFERENCES api_requests(id),
		FOREIGN KEY (proposed_policy_id) REFERENCES policies(id) ON DELETE SET NULL
	);`

	// Documents associations (for both APIs and requests)
	// Note: We'll use core.Document for the actual document storage
	documentAssociationsTable := `
	CREATE TABLE IF NOT EXISTS document_associations (
		id TEXT PRIMARY KEY,                          -- UUID for association
		document_filename TEXT NOT NULL,              -- Reference to document filename in chromem
		entity_id TEXT NOT NULL,                     -- Can be API ID or request ID
		entity_type TEXT NOT NULL CHECK (entity_type IN ('api', 'request')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (document_filename, entity_id, entity_type)
	);`

	// API external user access permissions
	apiUserAccessTable := `
	CREATE TABLE IF NOT EXISTS api_user_access (
		id TEXT PRIMARY KEY,                          -- UUID for access record
		api_id TEXT NOT NULL,
		external_user_id TEXT NOT NULL,
		access_level TEXT NOT NULL DEFAULT 'read' CHECK (access_level IN ('read', 'write', 'admin')),
		granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		granted_by TEXT,                             -- Always references the host user
		revoked_at DATETIME,
		is_active BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE,
		UNIQUE (api_id, external_user_id)
	);`

	// Trackers table (for referencing required trackers)
	trackersTable := `
	CREATE TABLE IF NOT EXISTS trackers (
		id TEXT PRIMARY KEY,                          -- UUID for tracker
		name TEXT NOT NULL,
		description TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Association table for requests and required trackers
	requestTrackersTable := `
	CREATE TABLE IF NOT EXISTS request_required_trackers (
		id TEXT PRIMARY KEY,                          -- UUID for association
		request_id TEXT NOT NULL,
		tracker_id TEXT NOT NULL,
		FOREIGN KEY (request_id) REFERENCES api_requests(id) ON DELETE CASCADE,
		FOREIGN KEY (tracker_id) REFERENCES trackers(id) ON DELETE CASCADE,
		UNIQUE (request_id, tracker_id)
	);`

	// Flexible policy system
	policiesTable := `
	CREATE TABLE IF NOT EXISTS policies (
		id TEXT PRIMARY KEY,                          -- UUID for policy
		name TEXT NOT NULL,
		description TEXT,
		type TEXT NOT NULL,                          -- 'token', 'time', 'credit', 'rate', 'composite', 'free'
		is_active BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT                               -- Always references the host user
	);`

	// Policy rules for complex policies
	policyRulesTable := `
	CREATE TABLE IF NOT EXISTS policy_rules (
		id TEXT PRIMARY KEY,                          -- UUID for rule
		policy_id TEXT NOT NULL,
		rule_type TEXT NOT NULL,                     -- 'token', 'time', 'credit', 'rate', etc.
		limit_value REAL,                            -- Numerical limit (tokens, minutes, credits)
		period TEXT,                                 -- 'minute', 'hour', 'day', 'week', 'month', 'year'
		action TEXT NOT NULL,                        -- 'block', 'throttle', 'notify', 'log'
		priority INTEGER DEFAULT 100,                 -- Priority for rule processing
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE
	);`

	// API usage tracking for policy enforcement
	apiUsageTable := `
	CREATE TABLE IF NOT EXISTS api_usage (
		id TEXT PRIMARY KEY,                          -- UUID for usage record
		api_id TEXT NOT NULL,
		external_user_id TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		request_count INTEGER DEFAULT 1,
		tokens_used INTEGER DEFAULT 0,
		credits_consumed REAL DEFAULT 0,
		execution_time_ms INTEGER DEFAULT 0,
		endpoint TEXT,
		was_throttled BOOLEAN DEFAULT FALSE,
		was_blocked BOOLEAN DEFAULT FALSE,
		FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE
	);`

	// Create usage summary table (updated periodically)
	apiUsageSummaryTable := `
	CREATE TABLE IF NOT EXISTS api_usage_summary (
		id TEXT PRIMARY KEY,                          -- UUID for summary record
		api_id TEXT NOT NULL,
		external_user_id TEXT NOT NULL,
		period_type TEXT NOT NULL,                    -- 'daily', 'weekly', 'monthly'
		period_start DATETIME NOT NULL,
		period_end DATETIME NOT NULL,
		total_requests INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		total_credits REAL DEFAULT 0,
		total_time_ms INTEGER DEFAULT 0,
		throttled_requests INTEGER DEFAULT 0,
		blocked_requests INTEGER DEFAULT 0,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE,
		UNIQUE (api_id, external_user_id, period_type, period_start)
	);`

	// Policy change history
	policyChangesTable := `
	CREATE TABLE IF NOT EXISTS policy_changes (
		id TEXT PRIMARY KEY,                          -- UUID for change record
		api_id TEXT NOT NULL,
		old_policy_id TEXT,
		new_policy_id TEXT,
		changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		changed_by TEXT,                              -- Always references the host user
		effective_date DATETIME,
		change_reason TEXT,
		FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE,
		FOREIGN KEY (old_policy_id) REFERENCES policies(id) ON DELETE SET NULL,
		FOREIGN KEY (new_policy_id) REFERENCES policies(id) ON DELETE SET NULL
	);`

	// Notifications table for quota alerts
	quotaNotificationsTable := `
	CREATE TABLE IF NOT EXISTS quota_notifications (
		id TEXT PRIMARY KEY,                          -- UUID for notification
		api_id TEXT NOT NULL,
		external_user_id TEXT NOT NULL,
		notification_type TEXT NOT NULL,              -- 'approaching_limit', 'limit_reached', 'policy_changed'
		rule_type TEXT,                               -- 'token', 'time', 'credit', 'rate'
		percentage_used REAL,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_read BOOLEAN DEFAULT FALSE,
		read_at DATETIME,
		FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE
	);`

	// Execute all table creation statements
	tables := []struct {
		name  string
		query string
	}{
		{"policies", policiesTable},
		{"apis", apisTable},
		{"api_requests", apiRequestsTable},
		{"document_associations", documentAssociationsTable},
		{"api_user_access", apiUserAccessTable},
		{"trackers", trackersTable},
		{"request_required_trackers", requestTrackersTable},
		{"policy_rules", policyRulesTable},
		{"api_usage", apiUsageTable},
		{"api_usage_summary", apiUsageSummaryTable},
		{"policy_changes", policyChangesTable},
		{"quota_notifications", quotaNotificationsTable},
	}

	for _, table := range tables {
		if _, err := db.Exec(table.query); err != nil {
			return fmt.Errorf("failed to create %s table: %v", table.name, err)
		}
	}

	return nil
}
