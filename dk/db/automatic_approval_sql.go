package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// InsertRule adds a brand‑new automatic approval rule.
func InsertRule(ctx context.Context, db *sql.DB, rule string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO automatic_approval_rules (rule) VALUES (?)`, rule)
	if err != nil {
		// UNIQUE constraint → give a cleaner error upstream
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("rule already exists")
		}
		return fmt.Errorf("insert rule: %w", err)
	}
	return nil
}

// DeleteRule removes a rule, returns <true> when something was deleted.
func DeleteRule(ctx context.Context, db *sql.DB, rule string) (bool, error) {
	res, err := db.ExecContext(ctx,
		`DELETE FROM automatic_approval_rules WHERE rule = ?`, rule)
	if err != nil {
		return false, fmt.Errorf("delete rule: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ListRules returns every rule, newest first.
func ListRules(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT rule FROM automatic_approval_rules ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
