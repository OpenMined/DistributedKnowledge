package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Query is your existing struct (unchanged)
type Query struct {
	ID               string   `json:"id"`
	From             string   `json:"from"`
	Question         string   `json:"question"`
	Answer           string   `json:"answer,omitempty"`
	DocumentsRelated []string `json:"documents_related"`
	Status           string   `json:"status"`
	Reason           string   `json:"reason,omitempty"`
}

// --- Helpers ---------------------------------------------------------------

// Insert a brand‑new query row.
func InsertQuery(ctx context.Context, db *sql.DB, q Query) error {
	docs, _ := json.Marshal(q.DocumentsRelated)
	_, err := db.ExecContext(ctx,
		`INSERT INTO queries 
		 (id, from_source, question, answer, documents_related, status, reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		q.ID, q.From, q.Question, q.Answer, string(docs), q.Status, q.Reason)
	if err != nil {
		return fmt.Errorf("insert query: %w", err)
	}
	return nil
}

// Fetch all (optionally filtered) queries.
func ListQueries(ctx context.Context, db *sql.DB, status, from string) ([]Query, error) {
	query := `SELECT id, from_source, question, answer, documents_related, status, reason 
	          FROM queries`
	var args []any
	var where []string
	if status != "" {
		where = append(where, "LOWER(status)=LOWER(?)")
		args = append(args, status)
	}
	if from != "" {
		where = append(where, "from_source=?")
		args = append(args, from)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list queries: %w", err)
	}
	defer rows.Close()

	var out []Query
	for rows.Next() {
		var q Query
		var docs string
		if err := rows.Scan(&q.ID, &q.From, &q.Question, &q.Answer,
			&docs, &q.Status, &q.Reason); err != nil {
			return nil, fmt.Errorf("scan query row: %w", err)
		}
		_ = json.Unmarshal([]byte(docs), &q.DocumentsRelated)
		out = append(out, q)
	}
	return out, rows.Err()
}

// Update status only, returns sql.ErrNoRows if nothing updated.
func UpdateQueryStatus(ctx context.Context, db *sql.DB, id, status string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE queries SET status=? WHERE id=?`, status, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Get one query by id.
func GetQuery(ctx context.Context, db *sql.DB, id string) (Query, error) {
	var q Query
	var docs string
	err := db.QueryRowContext(ctx,
		`SELECT id, from_source, question, answer, documents_related, status, reason
		 FROM queries WHERE id=?`, id).
		Scan(&q.ID, &q.From, &q.Question, &q.Answer, &docs, &q.Status, &q.Reason)
	if err != nil {
		return q, err
	}
	_ = json.Unmarshal([]byte(docs), &q.DocumentsRelated)
	return q, nil
}
