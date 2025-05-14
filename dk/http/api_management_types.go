package http

import (
	"time"
)

// User Access Management Types

// APIUserListQueryParams represents the query parameters for filtering API users
type APIUserListQueryParams struct {
	Active bool   `json:"active"` // Filter by active status
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`  // "granted_at", "user_id"
	Order  string `json:"order"` // "asc", "desc"
}

// APIUserListResponse represents the response for GET /api/apis/:id/users
type APIUserListResponse struct {
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Users  []APIUserAccess `json:"users"`
}

// APIUserAccess represents external user access to an API
type APIUserAccess struct {
	ID          string     `json:"id"`
	APIID       string     `json:"api_id"`
	UserID      string     `json:"user_id"`
	UserDetails UserRef    `json:"user_details,omitempty"`
	AccessLevel string     `json:"access_level"`
	GrantedAt   time.Time  `json:"granted_at"`
	GrantedBy   string     `json:"granted_by,omitempty"`
	IsActive    bool       `json:"is_active"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

// APIUserAccessRequest represents the request body for POST /api/apis/:id/users
type APIUserAccessRequest struct {
	UserID      string `json:"user_id"`
	AccessLevel string `json:"access_level"` // "read", "write", "admin"
}

// APIUserAccessUpdateRequest represents the request body for PATCH /api/apis/:id/users/:user_id
type APIUserAccessUpdateRequest struct {
	AccessLevel string `json:"access_level"` // "read", "write", "admin"
}

// APIUserAccessResponse represents the response for user access operations
type APIUserAccessResponse struct {
	ID          string     `json:"id"`
	APIID       string     `json:"api_id"`
	UserID      string     `json:"user_id"`
	AccessLevel string     `json:"access_level"`
	IsActive    bool       `json:"is_active"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

// API Entity Endpoints Types

// APIListQueryParams represents the query parameters for filtering APIs
type APIListQueryParams struct {
	Status         string `json:"status"` // "active", "inactive", "deprecated"
	ExternalUserID string `json:"external_user_id"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	Sort           string `json:"sort"`  // "name", "created_at"
	Order          string `json:"order"` // "asc", "desc"
}

// APIListResponse represents the response for GET /api/apis
type APIListResponse struct {
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	APIs   []APIBasic `json:"apis"`
}

// APIBasic represents the simplified API information returned in lists
type APIBasic struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	IsActive           bool       `json:"is_active"`
	IsDeprecated       bool       `json:"is_deprecated"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Policy             *PolicyRef `json:"policy,omitempty"`
	ExternalUsersCount int        `json:"external_users_count"`
	DocumentsCount     int        `json:"documents_count"`
}

// PolicyRef provides a simple reference to a policy
type PolicyRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// APIDetailResponse represents the response for GET /api/apis/:id
type APIDetailResponse struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	IsActive      bool          `json:"is_active"`
	IsDeprecated  bool          `json:"is_deprecated"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	APIKey        string        `json:"api_key"`
	ExternalUsers []UserRef     `json:"external_users"`
	Documents     []DocumentRef `json:"documents"`
	Policy        *PolicyDetail `json:"policy,omitempty"`
	UsageSummary  *UsageSummary `json:"usage_summary,omitempty"`
}

// UserRef provides a simple reference to a user
type UserRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	AccessLevel string `json:"access_level"`
}

// DocumentRef provides a simple reference to a document
type DocumentRef struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	UploadedAt time.Time `json:"uploaded_at"`
	SizeBytes  int       `json:"size_bytes"`
}

// PolicyDetail includes the policy rules
type PolicyDetail struct {
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Type  string             `json:"type"`
	Rules []PolicyRuleDetail `json:"rules"`
}

// PolicyRuleDetail represents a policy rule
type PolicyRuleDetail struct {
	Type   string  `json:"type"`
	Limit  float64 `json:"limit"`
	Period string  `json:"period,omitempty"`
	Action string  `json:"action"`
}

// UsageSummary provides API usage statistics
type UsageSummary struct {
	Today struct {
		Requests          int `json:"requests"`
		Tokens            int `json:"tokens"`
		ThrottledRequests int `json:"throttled_requests"`
		BlockedRequests   int `json:"blocked_requests"`
	} `json:"today"`
	ThisMonth struct {
		Requests          int `json:"requests"`
		Tokens            int `json:"tokens"`
		ThrottledRequests int `json:"throttled_requests"`
		BlockedRequests   int `json:"blocked_requests"`
	} `json:"this_month"`
}

// CreateAPIRequest represents the request body for POST /api/apis
type CreateAPIRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PolicyID      string   `json:"policy_id"`
	DocumentIDs   []string `json:"document_ids"`
	ExternalUsers []struct {
		UserID      string `json:"user_id"`
		AccessLevel string `json:"access_level"`
	} `json:"external_users"`
	IsActive bool `json:"is_active"`
}

// UpdateAPIRequest represents the request body for PATCH /api/apis/:id
type UpdateAPIRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PolicyID    *string `json:"policy_id,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// DeprecateAPIRequest represents the request body for POST /api/apis/:id/deprecate
type DeprecateAPIRequest struct {
	DeprecationDate    time.Time `json:"deprecation_date"`
	DeprecationMessage string    `json:"deprecation_message"`
}

// SuccessResponse is a simple response for successful operations
type SuccessResponse struct {
	Status string `json:"status"`
}

// API Request Endpoints Types

// APIRequestListQueryParams represents the query parameters for filtering requests
type APIRequestListQueryParams struct {
	Status      string `json:"status"`       // "pending", "approved", "denied"
	RequesterID string `json:"requester_id"` // External user ID
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	Sort        string `json:"sort"`  // "submitted_date", "api_name"
	Order       string `json:"order"` // "asc", "desc"
}

// APIRequestListResponse represents the response for GET /api/requests
type APIRequestListResponse struct {
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
	Requests []APIRequestBasic `json:"requests"`
}

// APIRequestBasic represents the simplified request information returned in lists
type APIRequestBasic struct {
	ID                    string    `json:"id"`
	APIName               string    `json:"api_name"`
	Description           string    `json:"description"`
	Status                string    `json:"status"` // "pending", "approved", "denied"
	SubmissionCount       int       `json:"submission_count"`
	SubmittedDate         time.Time `json:"submitted_date"`
	Requester             UserRef   `json:"requester"`
	DocumentsCount        int       `json:"documents_count"`
	RequiredTrackersCount int       `json:"required_trackers_count"`
}

// APIRequestDetailResponse represents the response for GET /api/requests/:id
type APIRequestDetailResponse struct {
	ID               string         `json:"id"`
	APIName          string         `json:"api_name"`
	Description      string         `json:"description"`
	SubmittedDate    time.Time      `json:"submitted_date"`
	Status           string         `json:"status"`
	Requester        UserRef        `json:"requester"`
	Documents        []DocumentRef  `json:"documents"`
	RequiredTrackers []TrackerRef   `json:"required_trackers"`
	DenialReason     string         `json:"denial_reason,omitempty"`
	DeniedDate       *time.Time     `json:"denied_date,omitempty"`
	ApprovedDate     *time.Time     `json:"approved_date,omitempty"`
	SubmissionCount  int            `json:"submission_count"`
	PreviousRequest  *APIRequestRef `json:"previous_request,omitempty"`
	ProposedPolicy   *PolicyRef     `json:"proposed_policy,omitempty"`
}

// TrackerRef provides a simple reference to a tracker
type TrackerRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// APIRequestRef provides a reference to a previous request
type APIRequestRef struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	SubmittedDate time.Time `json:"submitted_date"`
}

// CreateAPIRequestRequest represents the request body for POST /api/requests
type CreateAPIRequestRequest struct {
	APIName            string   `json:"api_name"`
	Description        string   `json:"description"`
	DocumentIDs        []string `json:"document_ids"`
	RequiredTrackerIDs []string `json:"required_tracker_ids"`
	ProposedPolicyID   string   `json:"proposed_policy_id,omitempty"`
}

// UpdateAPIRequestStatusRequest represents the request body for PATCH /api/requests/:id/status
type UpdateAPIRequestStatusRequest struct {
	Status       string `json:"status"`                  // "approved" or "denied"
	PolicyID     string `json:"policy_id,omitempty"`     // Required if status is "approved"
	CreateAPI    bool   `json:"create_api,omitempty"`    // Whether to automatically create an API
	DenialReason string `json:"denial_reason,omitempty"` // Required if status is "denied"
}

// ResubmitAPIRequestRequest represents the request body for POST /api/requests/:id/resubmit
type ResubmitAPIRequestRequest struct {
	Description        string   `json:"description,omitempty"`          // Updated description
	DocumentIDs        []string `json:"document_ids,omitempty"`         // Updated document IDs
	RequiredTrackerIDs []string `json:"required_tracker_ids,omitempty"` // Updated tracker IDs
	ProposedPolicyID   string   `json:"proposed_policy_id,omitempty"`   // Updated policy ID
}

// Document Endpoints Types

// DocumentListQueryParams represents the query parameters for filtering documents
type DocumentListQueryParams struct {
	EntityType string `json:"entity_type"` // "api" or "request"
	EntityID   string `json:"entity_id"`   // API or request ID
	IsDeleted  bool   `json:"is_deleted"`  // Include deleted documents
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// DocumentListResponse represents the response for GET /api/documents
type DocumentListResponse struct {
	Total     int           `json:"total"`
	Limit     int           `json:"limit"`
	Offset    int           `json:"offset"`
	Documents []DocumentRef `json:"documents"`
}

// DocumentDetailResponse represents the response for GET /api/documents/{id}
type DocumentDetailResponse struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	ContentType  string              `json:"content_type,omitempty"`
	SizeBytes    int                 `json:"size_bytes"`
	Content      string              `json:"content"`
	UploadedAt   time.Time           `json:"uploaded_at"`
	UploaderID   string              `json:"uploader_id,omitempty"`
	IsDeleted    bool                `json:"is_deleted"`
	DeletionDate *time.Time          `json:"deletion_date,omitempty"`
	Associations []EntityAssociation `json:"associations"`
	Metadata     map[string]string   `json:"metadata,omitempty"`
}

// EntityAssociation represents an association between a document and an entity
type EntityAssociation struct {
	EntityID   string    `json:"entity_id"`
	EntityType string    `json:"entity_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// DocumentAssociateRequest represents the request body for POST /api/documents/associate
type DocumentAssociateRequest struct {
	DocumentID string `json:"document_id"`
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
}

// Policy Management Types

// PolicyListQueryParams represents the query parameters for filtering policies
type PolicyListQueryParams struct {
	Type   string `json:"type"`   // Filter by policy type
	Active bool   `json:"active"` // Filter by active status
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`  // "name", "created_at", "type"
	Order  string `json:"order"` // "asc", "desc"
}

// PolicyListResponse represents the response for GET /api/policies
type PolicyListResponse struct {
	Total    int            `json:"total"`
	Limit    int            `json:"limit"`
	Offset   int            `json:"offset"`
	Policies []PolicyDetail `json:"policies"`
}

// CreatePolicyRequest represents the request body for POST /api/policies
type CreatePolicyRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"` // "free", "rate", "token", "time", "credit", "composite"
	Rules       []PolicyRule `json:"rules,omitempty"`
}

// UpdatePolicyRequest represents the request body for PATCH /api/policies/:id
type UpdatePolicyRequest struct {
	Name        *string      `json:"name,omitempty"`
	Description *string      `json:"description,omitempty"`
	IsActive    *bool        `json:"is_active,omitempty"`
	Rules       []PolicyRule `json:"rules,omitempty"`
}

// PolicyRule represents a single rule within a policy
type PolicyRule struct {
	RuleType   string  `json:"rule_type"` // "rate", "token", "time", "credit"
	LimitValue float64 `json:"limit_value"`
	Period     string  `json:"period,omitempty"` // "minute", "hour", "day", "week", "month", "year"
	Action     string  `json:"action"`           // "block", "throttle", "notify", "log"
	Priority   int     `json:"priority,omitempty"`
}

// ChangePolicyRequest represents the request body for POST /api/apis/:id/policy
type ChangePolicyRequest struct {
	PolicyID             string     `json:"policy_id"`
	EffectiveImmediately bool       `json:"effective_immediately"`
	ScheduledDate        *time.Time `json:"scheduled_date,omitempty"`
	ChangeReason         string     `json:"change_reason"`
}

// PolicyChangeResponse represents a policy change record
type PolicyChangeResponse struct {
	ID            string     `json:"id"`
	APIID         string     `json:"api_id"`
	OldPolicy     *PolicyRef `json:"old_policy,omitempty"`
	NewPolicy     *PolicyRef `json:"new_policy,omitempty"`
	ChangedAt     time.Time  `json:"changed_at"`
	ChangedBy     string     `json:"changed_by,omitempty"`
	EffectiveDate *time.Time `json:"effective_date,omitempty"`
	ChangeReason  string     `json:"change_reason,omitempty"`
}

// PolicyChangeHistoryResponse represents the response for GET /api/apis/:id/policy/history
type PolicyChangeHistoryResponse struct {
	APIID   string                 `json:"api_id"`
	Changes []PolicyChangeResponse `json:"changes"`
}
