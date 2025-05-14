package db

import (
	"database/sql"
	"fmt"
	"os"
)

// DB is a wrapper around *sql.DB that adds some helper methods for tests
// This can be extended with more functionality as needed
type DB struct {
	*sql.DB
	Path string // Path to the database file (for cleanup)
}

// OpenTestDB creates a temporary SQLite database for testing
func OpenTestDB() (*DB, error) {
	// Create a temporary file for the test database
	tmpFile, err := os.CreateTemp("", "test_dk_*.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()

	// Initialize the database
	sqlDB, err := Initialize(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to initialize test database: %w", err)
	}

	// Return the wrapped DB
	return &DB{
		DB:   sqlDB,
		Path: tmpFile.Name(),
	}, nil
}

// Close closes the database connection and removes the temporary file
func (db *DB) Close() error {
	if db.DB != nil {
		err := db.DB.Close()
		if err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	// Remove the temporary file
	if db.Path != "" {
		err := os.Remove(db.Path)
		if err != nil {
			return fmt.Errorf("failed to remove database file: %w", err)
		}
	}

	return nil
}

// InitAPIManagementTables initializes API management tables for testing
func InitAPIManagementTables(db *sql.DB) error {
	// Run migrations for API management tables
	return RunAPIMigrations(db)
}
