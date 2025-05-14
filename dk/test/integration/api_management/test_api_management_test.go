package integration_test

import (
	"database/sql"
	"dk/test/utils"
	_ "modernc.org/sqlite"
	"testing"
)

func TestAPIManagement(t *testing.T) {
	// Open an in-memory database for testing
	db, err := sql.Open("sqlite", ":memory:?_busy_timeout=5000&_journal_mode=DELETE&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run API migrations
	t.Log("Running API Management schema migrations...")
	if err := utils.RunAPIMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify the tables were created
	t.Log("Verifying tables were created...")
	failures := utils.VerifyAPITables(db)

	if len(failures) > 0 {
		for _, tableName := range failures {
			t.Errorf("Table %s was not created properly", tableName)
		}
		t.Fail()
	} else {
		t.Log("All API Management tables were created successfully!")
	}
}
