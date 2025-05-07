package models

import "time"

// Message type constants
const (
	MessageTypeForward            = "forward"
	MessageTypeRegisterDocument   = "register_document"
	MessageTypeAppendDocument     = "append_document"
	MessageTypeRegisterDocSuccess = "register_document_success"
	MessageTypeRegisterDocError   = "register_document_error"
)

// User represents a registered user.
type User struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a message sent between users.
type Message struct {
	ID               int       `json:"id"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Timestamp        time.Time `json:"timestamp"`
	Content          string    `json:"content"`
	Status           string    `json:"status"` // e.g., "pending", "delivered", "verified"
	IsBroadcast      bool      `json:"is_broadcast,omitempty"`
	Signature        string    `json:"signature,omitempty"`          // Base64-encoded signature of message content
	IsForwardMessage bool      `json:"is_forward_message,omitempty"` // Indicates if this is a forward message
}

// TrackerDocuments represents the structure for tracker documents
type TrackerDocuments struct {
	Datasets  map[string]string `json:"datasets,omitempty"`
	Templates map[string]string `json:"templates,omitempty"`
}

// Tracker represents a user's tracker information.
type Tracker struct {
	ID                 int              `json:"id,omitempty"`
	UserID             string           `json:"user_id"`
	TrackerName        string           `json:"tracker_name"`
	TrackerDescription string           `json:"tracker_description,omitempty"`
	TrackerVersion     string           `json:"tracker_version,omitempty"`
	TrackerDocuments   TrackerDocuments `json:"tracker_documents,omitempty"`
	CreatedAt          time.Time        `json:"created_at,omitempty"`
	UpdatedAt          time.Time        `json:"updated_at,omitempty"`
}

// TrackerData represents the data for a single tracker without its name
type TrackerData struct {
	TrackerDescription string           `json:"tracker_description,omitempty"`
	TrackerVersion     string           `json:"tracker_version,omitempty"`
	TrackerDocuments   TrackerDocuments `json:"tracker_documents,omitempty"`
}

// TrackerListPayload represents the structure for updating all of a user's trackers
type TrackerListPayload struct {
	UserID   string                 `json:"user_id"`
	Trackers map[string]TrackerData `json:"trackers"` // Map of tracker name to tracker data
}

// API represents a user's API information.
type API struct {
	ID        int       `json:"id,omitempty"`
	UserID    string    `json:"user_id"`
	APIName   string    `json:"api_name"`
	Documents []string  `json:"documents,omitempty"`
	Policy    string    `json:"policy,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
