package models

import "time"

// User represents a registered user.
type User struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a message sent between users.
type Message struct {
	ID          int       `json:"id"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Timestamp   time.Time `json:"timestamp"`
	Content     string    `json:"content"`
	Status      string    `json:"status"` // e.g., "pending", "delivered", "verified"
	IsBroadcast bool      `json:"is_broadcast,omitempty"`
	Signature   string    `json:"signature,omitempty"` // Base64-encoded signature of message content
}
