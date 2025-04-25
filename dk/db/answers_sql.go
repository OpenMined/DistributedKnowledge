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
	rows, err := db.QueryContext(ctx,
		`SELECT user, answer FROM answers WHERE question = ? ORDER BY created_at ASC`, qID)
	if err != nil {
		return nil, fmt.Errorf("query answers: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var user, ans string
		if err := rows.Scan(&user, &ans); err != nil {
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		out[user] = ans
	}
	return out, rows.Err()
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
