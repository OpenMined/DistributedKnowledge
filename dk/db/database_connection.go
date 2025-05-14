package db

import (
	"database/sql"
)

// DatabaseConnection represents a connection to the database with additional state
type DatabaseConnection struct {
	DB          *sql.DB
	IsThrottled bool // Flag for tracking request throttling
}
