package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite database for testing
	dsn := ":memory:?_busy_timeout=5000&_journal_mode=DELETE&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
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
			t.Fatalf("Failed to set pragma (%s): %v", pragma, err)
		}
	}

	return db
}
