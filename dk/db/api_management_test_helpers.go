package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

// Use a global shared database for in-memory SQLite tests
var (
	sharedTestDB  *sql.DB
	dbInitialized bool
)

// setupTestDB creates an in-memory database for testing and runs all migrations
// It uses a shared connection to ensure all tests can see the same tables
func setupTestDB(t *testing.T) *sql.DB {
	// If we already have a shared test DB, return it
	if dbInitialized && sharedTestDB != nil {
		return sharedTestDB
	}

	// Use file-based SQLite database for testing since in-memory
	// databases cannot be shared between connections in SQLite
	dsn := "file::memory:?cache=shared&mode=memory"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Set pragmas for better performance and reliability
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA cache_size = 1000;",
		"PRAGMA foreign_keys = ON;", // This is crucial for foreign key constraints to work
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("Failed to set pragma (%s): %v", pragma, err)
		}
	}

	// Run the migrations to create all tables
	if err := RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations during setup: %v", err)
	}

	// Verify tables were created
	tables := []string{"apis", "api_requests", "document_associations", "api_user_access",
		"trackers", "request_required_trackers", "policies", "policy_rules",
		"api_usage", "api_usage_summary", "policy_changes", "quota_notifications"}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil || name == "" {
			t.Fatalf("Failed to create table %s during setup: %v", table, err)
		}
	}

	// Save the shared database for reuse
	sharedTestDB = db
	dbInitialized = true

	// Clear any existing data to ensure a clean state
	cleanTestTables(db)

	return db
}

// cleanTestTables clears all data from the test tables to ensure a clean state
func cleanTestTables(db *sql.DB) {
	tables := []string{
		"quota_notifications",
		"policy_changes",
		"api_usage_summary",
		"api_usage",
		"request_required_trackers",
		"document_associations",
		"api_user_access",
		"policy_rules",
		"apis",
		"api_requests",
		"trackers",
		"policies",
	}

	// Temporarily disable foreign key constraints
	_, _ = db.Exec("PRAGMA foreign_keys = OFF;")

	// Delete data from all tables in reverse order to avoid foreign key issues
	for _, table := range tables {
		_, _ = db.Exec("DELETE FROM " + table + ";")
	}

	// Re-enable foreign key constraints
	_, _ = db.Exec("PRAGMA foreign_keys = ON;")
}
