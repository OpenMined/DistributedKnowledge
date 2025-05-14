package db

import (
	"time"
)

// API represents an API entity in the system
type API struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Description        string     `json:"description,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	IsActive           bool       `json:"is_active"`
	APIKey             string     `json:"api_key,omitempty"`
	HostUserID         string     `json:"host_user_id"`
	PolicyID           *string    `json:"policy_id,omitempty"`
	IsDeprecated       bool       `json:"is_deprecated"`
	DeprecationDate    *time.Time `json:"deprecation_date,omitempty"`
	DeprecationMessage string     `json:"deprecation_message,omitempty"`
}

// APIRequest represents a request for API access
type APIRequest struct {
	ID                string     `json:"id"`
	APIName           string     `json:"api_name"`
	Description       string     `json:"description,omitempty"`
	SubmittedDate     time.Time  `json:"submitted_date"`
	Status            string     `json:"status"` // 'pending', 'approved', 'denied'
	RequesterID       string     `json:"requester_id"`
	DenialReason      string     `json:"denial_reason,omitempty"`
	DeniedDate        *time.Time `json:"denied_date,omitempty"`
	ApprovedDate      *time.Time `json:"approved_date,omitempty"`
	SubmissionCount   int        `json:"submission_count"`
	PreviousRequestID *string    `json:"previous_request_id,omitempty"`
	ProposedPolicyID  *string    `json:"proposed_policy_id,omitempty"`
}

// DocumentAssociation represents an association between a document and an API or request
type DocumentAssociation struct {
	ID               string    `json:"id"`
	DocumentFilename string    `json:"document_filename"`
	EntityID         string    `json:"entity_id"`
	EntityType       string    `json:"entity_type"` // 'api' or 'request'
	CreatedAt        time.Time `json:"created_at"`
}

// APIUserAccess represents access permissions for a user to an API
type APIUserAccess struct {
	ID             string     `json:"id"`
	APIID          string     `json:"api_id"`
	ExternalUserID string     `json:"external_user_id"`
	AccessLevel    string     `json:"access_level"` // 'read', 'write', 'admin'
	GrantedAt      time.Time  `json:"granted_at"`
	GrantedBy      string     `json:"granted_by,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	IsActive       bool       `json:"is_active"`
}

// Tracker represents a tracker that can be required for API requests
type Tracker struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// RequestRequiredTracker represents an association between a request and a required tracker
type RequestRequiredTracker struct {
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	TrackerID string `json:"tracker_id"`
}

// Policy represents a usage policy for APIs
type Policy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Type        string       `json:"type"` // 'token', 'time', 'credit', 'rate', 'composite', 'free'
	IsActive    bool         `json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CreatedBy   string       `json:"created_by,omitempty"`
	Rules       []PolicyRule `json:"rules,omitempty"` // Not stored in the DB, joined when needed
}

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

// APIUsage represents a single usage record for an API
type APIUsage struct {
	ID              string    `json:"id"`
	APIID           string    `json:"api_id"`
	ExternalUserID  string    `json:"external_user_id"`
	Timestamp       time.Time `json:"timestamp"`
	RequestCount    int       `json:"request_count"`
	TokensUsed      int       `json:"tokens_used"`
	CreditsConsumed float64   `json:"credits_consumed"`
	ExecutionTimeMs int       `json:"execution_time_ms"`
	Endpoint        string    `json:"endpoint,omitempty"`
	WasThrottled    bool      `json:"was_throttled"`
	WasBlocked      bool      `json:"was_blocked"`
}

// APIUsageSummary represents aggregated usage data for reporting
type APIUsageSummary struct {
	ID                string    `json:"id"`
	APIID             string    `json:"api_id"`
	ExternalUserID    string    `json:"external_user_id"`
	PeriodType        string    `json:"period_type"` // 'daily', 'weekly', 'monthly'
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
	TotalRequests     int       `json:"total_requests"`
	TotalTokens       int       `json:"total_tokens"`
	TotalCredits      float64   `json:"total_credits"`
	TotalTimeMs       int       `json:"total_time_ms"`
	ThrottledRequests int       `json:"throttled_requests"`
	BlockedRequests   int       `json:"blocked_requests"`
	LastUpdated       time.Time `json:"last_updated"`
}

// PolicyChange represents a history record of policy changes for an API
type PolicyChange struct {
	ID            string     `json:"id"`
	APIID         string     `json:"api_id"`
	OldPolicyID   *string    `json:"old_policy_id,omitempty"`
	NewPolicyID   *string    `json:"new_policy_id,omitempty"`
	ChangedAt     time.Time  `json:"changed_at"`
	ChangedBy     string     `json:"changed_by,omitempty"`
	EffectiveDate *time.Time `json:"effective_date,omitempty"`
	ChangeReason  string     `json:"change_reason,omitempty"`
}

// QuotaNotification represents a notification about policy usage
type QuotaNotification struct {
	ID               string     `json:"id"`
	APIID            string     `json:"api_id"`
	ExternalUserID   string     `json:"external_user_id"`
	NotificationType string     `json:"notification_type"`   // 'approaching_limit', 'limit_reached', 'policy_changed'
	RuleType         string     `json:"rule_type,omitempty"` // 'token', 'time', 'credit', 'rate'
	PercentageUsed   float64    `json:"percentage_used,omitempty"`
	Message          string     `json:"message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	IsRead           bool       `json:"is_read"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
}
