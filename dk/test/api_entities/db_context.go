package api_entities

import (
	"context"
	"database/sql"
	"errors"
)

// DBFromContext extracts the database connection from the context
func DBFromContext(ctx context.Context) (*sql.DB, error) {
	db, ok := ctx.Value("db").(*sql.DB)
	if !ok || db == nil {
		return nil, errors.New("database connection not found in context")
	}
	return db, nil
}

// Mock implementation of GetPolicy to retrieve a policy with its ID
func GetPolicy(db *sql.DB, policyID string) (*Policy, error) {
	policy := &Policy{
		ID:   policyID,
		Name: "Mock Policy",
		Type: "free",
	}
	return policy, nil
}

// Mock implementation of GetPolicyWithRules to retrieve a policy with rules
func GetPolicyWithRules(db *sql.DB, policyID string) (*Policy, error) {
	policy := &Policy{
		ID:   policyID,
		Name: "Mock Policy",
		Type: "free",
		Rules: []PolicyRule{
			{
				ID:         "rule-1",
				PolicyID:   policyID,
				RuleType:   "token",
				LimitValue: 1000,
				Period:     "day",
				Action:     "block",
				Priority:   1,
			},
		},
	}
	return policy, nil
}

// Mock Policy structure
type Policy struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Type  string       `json:"type"`
	Rules []PolicyRule `json:"rules,omitempty"`
}
