package api_entities

import (
	"time"
)

// PolicyRule represents a rule within a policy
type PolicyRule struct {
	ID         string    `json:"id"`
	PolicyID   string    `json:"policy_id"`
	RuleType   string    `json:"rule_type"` // 'token', 'time', 'credit', 'rate'
	LimitValue float64   `json:"limit_value,omitempty"`
	Period     string    `json:"period,omitempty"` // 'minute', 'hour', 'day', 'week', 'month', 'year'
	Action     string    `json:"action"`           // 'block', 'throttle', 'notify', 'log'
	Priority   int       `json:"priority"`
	CreatedAt  time.Time `json:"created_at"`
}
