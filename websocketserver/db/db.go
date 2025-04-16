package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Initialize opens a SQLite database connection and enables WAL mode.
func Initialize(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	// Enable Write-Ahead Logging mode for better concurrency.
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %v", err)
	}
	return db, nil
}

// RunMigrations creates the necessary tables if they do not exist.
func RunMigrations(db *sql.DB) error {
	descriptionsTable := `
	CREATE TABLE IF NOT EXISTS user_descriptions (
		user_id TEXT PRIMARY KEY,
		descriptions TEXT NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(user_id)
	);`

	sessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		duration INTEGER  -- duration in seconds
	);`

	messageEventsTable := `
	CREATE TABLE IF NOT EXISTS message_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		is_broadcast BOOLEAN NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(session_id) REFERENCES sessions(session_id)
	);`

	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		public_key TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	messageTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_user TEXT NOT NULL,
		to_user TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		content TEXT NOT NULL,
		status TEXT NOT NULL,
    is_broadcast BOOLEAN DEFAULT FALSE,
    signature TEXT,
		FOREIGN KEY(from_user) REFERENCES users(user_id),
		FOREIGN KEY(to_user) REFERENCES users(user_id)
	);`

	messageDeliveries := `
  CREATE TABLE IF NOT EXISTS broadcast_deliveries (
    message_id INT,
    user_id VARCHAR(255),
    delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id)
  );
  `
	if _, err := db.Exec(userTable); err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}
	if _, err := db.Exec(messageTable); err != nil {
		return fmt.Errorf("failed to create messages table: %v", err)
	}
	if _, err := db.Exec(messageDeliveries); err != nil {
		return fmt.Errorf("failed to create broadcast_deliveriestable: %v", err)
	}

	if _, err := db.Exec(sessionsTable); err != nil {
		return fmt.Errorf("failed to create sessions table: %v", err)
	}
	if _, err := db.Exec(messageEventsTable); err != nil {
		return fmt.Errorf("failed to create message_events table: %v", err)
	}

	if _, err := db.Exec(descriptionsTable); err != nil {
		return fmt.Errorf("failed to create user_descriptions table: %v", err)
	}
	return nil
}
