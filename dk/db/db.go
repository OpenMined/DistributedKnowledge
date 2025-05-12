package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

// Initialize opens a SQLite database connection and enables WAL mode.
func Initialize(dbPath string) (*sql.DB, error) {
	// Ensure the directory exists before opening the database
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Use a DSN with memory settings and timeout configurations
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=DELETE&cache=shared", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set pragmas for better performance and reliability
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA cache_size = 1000;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma (%s): %v", pragma, err)
		}
	}

	// Set connection limits
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// RunMigrations creates the necessary tables if they do not exist.
func RunMigrations(db *sql.DB) error {
	queriesTable := `
	CREATE TABLE IF NOT EXISTS queries (
		id                TEXT PRIMARY KEY,               -- "qry‑…" identifier
		from_source       TEXT  NOT NULL,                 -- maps the JSON key "from"
		question          TEXT  NOT NULL,
		answer            TEXT,                           -- may still be empty / NULL
		documents_related TEXT,                           -- store JSON array ([]string) as TEXT
		status            TEXT  NOT NULL,                 -- e.g. "pending", "accepted"
		reason            TEXT,
		created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Answers given by users to specific questions
	answersTable := `
	CREATE TABLE IF NOT EXISTS answers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		question      TEXT NOT NULL,
		user          TEXT NOT NULL,
		answer        TEXT NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (question, user)            -- avoid duplicate entries
	);`

	// New‑app requests awaiting manual or automatic approval
	appRequestsTable := `
	CREATE TABLE IF NOT EXISTS app_requests (
		app_name        TEXT PRIMARY KEY,          -- e.g. "cpu_tracker"
		requested_by    TEXT NOT NULL,             -- who asked for it
		app_description TEXT NOT NULL,
		status          TEXT NOT NULL,             -- "pending", "accepted", …
		reason          TEXT,
		safety          TEXT,                      -- "Undefined" etc.
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// List of automatic approval rules (simple strings for now)
	automaticApprovalTable := `
	CREATE TABLE IF NOT EXISTS automatic_approval_rules (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		rule    TEXT NOT NULL UNIQUE,              -- e.g. "Approve all"
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// General one‑line descriptions (not user‑specific)
	globalDescriptionsTable := `
	CREATE TABLE IF NOT EXISTS descriptions_global (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL UNIQUE,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(answersTable); err != nil {
		return fmt.Errorf("failed to create answers table: %v", err)
	}
	if _, err := db.Exec(appRequestsTable); err != nil {
		return fmt.Errorf("failed to create app_requests table: %v", err)
	}
	if _, err := db.Exec(automaticApprovalTable); err != nil {
		return fmt.Errorf("failed to create automatic_approval_rules table: %v", err)
	}
	if _, err := db.Exec(globalDescriptionsTable); err != nil {
		return fmt.Errorf("failed to create descriptions_global table: %v", err)
	}

	// new migration for the queries table
	if _, err := db.Exec(queriesTable); err != nil {
		return fmt.Errorf("failed to create queries table: %v", err)
	}
	return nil
}
