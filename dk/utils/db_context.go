package utils

import (
	"context"
	"database/sql"
	"errors"
)

// DBFromContext extracts the database connection from the context
// This will check both the databaseKey{} context key (used by handlers) and the "db" string key (used by tests)
func DBFromContext(ctx context.Context) (*sql.DB, error) {
	// First check for the databaseKey{} (used by handlers)
	if db, ok := ctx.Value(databaseKey{}).(*sql.DB); ok && db != nil {
		return db, nil
	}

	// Then check for the "db" string key (used by tests)
	if db, ok := ctx.Value("db").(*sql.DB); ok && db != nil {
		return db, nil
	}

	return nil, errors.New("database connection not found in context")
}

// UserIDFromContext extracts the user ID from the context
func UserIDFromContext(ctx context.Context) (string, error) {
	// First check for the UserIDContextKey (used by handlers)
	if userID, ok := ctx.Value(UserIDContextKey).(string); ok && userID != "" {
		return userID, nil
	}

	// Then check for the "user_id" string key (used by tests)
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		return userID, nil
	}

	return "", errors.New("user ID not found in context")
}

// LogError logs an error with a formatted message
func LogError(ctx context.Context, format string, args ...interface{}) {
	// No-op implementation for testing
}
