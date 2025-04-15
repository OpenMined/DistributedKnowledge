// Package metrics provides functions to persist engagement metrics in the database.
package metrics

import (
	"database/sql"
	"fmt"
	"time"
)

var db *sql.DB

// InitPersistence initializes the metrics persistence by saving the database connection.
// Call this from your main function after the DB is initialized.
func InitPersistence(database *sql.DB) {
	db = database
	if err := createMetricsTables(); err != nil {
		fmt.Printf("Failed to create metrics tables: %v\n", err)
	}
}

// createMetricsTables creates the metrics tables if they do not exist.
func createMetricsTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS message_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			is_broadcast BOOLEAN NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(session_id) REFERENCES sessions(session_id)
		);`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// RecordSessionStartPersist inserts a new session record into the sessions table.
func RecordSessionStartPersist(sessionID, userID string, startTime time.Time) {
	if db == nil {
		return
	}
	query := `INSERT INTO sessions(session_id, user_id, start_time) VALUES(?, ?, ?)`
	if _, err := db.Exec(query, sessionID, userID, startTime); err != nil {
		fmt.Printf("Error persisting session start: %v\n", err)
	}
}

// RecordSessionEndPersist updates the session record with the end time and duration.
func RecordSessionEndPersist(sessionID string, endTime time.Time) {
	if db == nil {
		return
	}
	var startTime time.Time
	query := `SELECT start_time FROM sessions WHERE session_id = ?`
	if err := db.QueryRow(query, sessionID).Scan(&startTime); err != nil {
		fmt.Printf("Error fetching session start time: %v\n", err)
		return
	}
	duration := int(endTime.Sub(startTime).Seconds())
	update := `UPDATE sessions SET end_time = ?, duration = ? WHERE session_id = ?`
	if _, err := db.Exec(update, endTime, duration, sessionID); err != nil {
		fmt.Printf("Error updating session record: %v\n", err)
	}
}

// RecordMessageEventPersist records each message event in the database.
func RecordMessageEventPersist(sessionID, userID string, isBroadcast bool, ts time.Time) {
	if db == nil {
		return
	}
	query := `INSERT INTO message_events(session_id, user_id, is_broadcast, timestamp) VALUES(?, ?, ?, ?)`
	if _, err := db.Exec(query, sessionID, userID, isBroadcast, ts); err != nil {
		fmt.Printf("Error persisting message event: %v\n", err)
	}
}
