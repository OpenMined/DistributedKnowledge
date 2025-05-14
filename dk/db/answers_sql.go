package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Answer mirrors one row of the `answers` table.
type Answer struct {
	Question  string    `json:"question"`   // query‑id
	User      string    `json:"user"`       // who answered
	Text      string    `json:"answer"`     // the answer itself
	CreatedAt time.Time `json:"created_at"` // filled by the DB
}

/*
   ──────────────────────────────────────────────────────────────────────────────
   WRITE helpers
*/

// InsertAnswer inserts a fresh answer or replaces an existing one (same
// question+user).  The UNIQUE(question,user) constraint defined in the
// migration lets us rely on `ON CONFLICT … DO UPDATE`.
func InsertAnswer(ctx context.Context, db *sql.DB, a Answer) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO answers (question, user, answer)
		VALUES (?, ?, ?)
		ON CONFLICT(question, user)
		DO UPDATE SET
		    answer      = excluded.answer,
		    created_at  = CURRENT_TIMESTAMP;`,
		a.Question, a.User, a.Text)
	if err != nil {
		return fmt.Errorf("insert answer: %w", err)
	}
	return nil
}

/*
   ──────────────────────────────────────────────────────────────────────────────
   READ helpers
*/

// AnswersForQuestion returns the map[user]answer for one query id.
func AnswersForQuestion(ctx context.Context, db *sql.DB, qID string) (map[string]string, error) {
	fmt.Printf("[SQL-DEBUG] Starting AnswersForQuestion for query ID: '%s'\n", qID)

	// Build the SQL query
	query := `SELECT user, answer FROM answers WHERE question = ? ORDER BY created_at ASC`
	fmt.Printf("[SQL-DEBUG] Executing SQL: '%s' with parameter: '%s'\n", query, qID)

	// Execute the query
	rows, err := db.QueryContext(ctx, query, qID)
	if err != nil {
		fmt.Printf("[SQL-ERROR] Failed to execute query: %v\n", err)
		return nil, fmt.Errorf("query answers: %w", err)
	}
	defer rows.Close()
	fmt.Println("[SQL-DEBUG] Query executed successfully")

	// Process the results
	out := make(map[string]string)
	rowCount := 0

	fmt.Println("[SQL-DEBUG] Processing result rows...")
	for rows.Next() {
		rowCount++
		var user, ans string
		if err := rows.Scan(&user, &ans); err != nil {
			fmt.Printf("[SQL-ERROR] Failed to scan row %d: %v\n", rowCount, err)
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		fmt.Printf("[SQL-DEBUG] Row %d: user='%s', answer length=%d\n", rowCount, user, len(ans))
		out[user] = ans
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		fmt.Printf("[SQL-ERROR] Error during row iteration: %v\n", err)
		return nil, fmt.Errorf("iterate answers: %w", err)
	}

	fmt.Printf("[SQL-DEBUG] AnswersForQuestion complete. Found %d answers for query ID '%s'\n", len(out), qID)
	return out, nil
}

// AllAnswers returns the nested map[question]map[user]answer.
func AllAnswers(ctx context.Context, db *sql.DB) (map[string]map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT question, user, answer FROM answers ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query all answers: %w", err)
	}
	defer rows.Close()

	out := make(map[string]map[string]string)
	for rows.Next() {
		var qID, user, ans string
		if err := rows.Scan(&qID, &user, &ans); err != nil {
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		if out[qID] == nil {
			out[qID] = make(map[string]string)
		}
		out[qID][user] = ans
	}
	return out, rows.Err()
}
