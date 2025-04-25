package auth

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// SecurityEvent represents a security-related event for audit logging
type SecurityEvent struct {
	Timestamp time.Time
	Event     string
	UserID    string
	IP        string
	Success   bool
	Details   string
}

// Logger handles security event logging
type Logger struct {
	// This could be expanded to include database logging, file logging, etc.
}

// NewLogger creates a new security logger
func NewLogger() *Logger {
	return &Logger{}
}

// LogAuthEvent logs an authentication event
func (l *Logger) LogAuthEvent(event SecurityEvent) {
	timestamp := event.Timestamp.Format(time.RFC3339)
	statusStr := "FAILED"
	if event.Success {
		statusStr = "SUCCESS"
	}

	logMessage := fmt.Sprintf(
		"[%s] SECURITY EVENT: %s | User: %s | IP: %s | Status: %s | %s",
		timestamp,
		event.Event,
		event.UserID,
		event.IP,
		statusStr,
		event.Details,
	)

	log.Println(logMessage)

	// TODO: In production, this should also log to a dedicated security log file
	// or database table for audit and compliance purposes
}

// GetClientIP extracts the client IP address from the request
// Properly handles reverse proxies by checking X-Forwarded-For header
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// The first IP in the list is the original client
		return forwarded
	}

	// If no X-Forwarded-For, use RemoteAddr
	return r.RemoteAddr
}

// Common security event type constants
const (
	EventLogin                = "LOGIN"
	EventTokenCreation        = "TOKEN_CREATION"
	EventTokenVerification    = "TOKEN_VERIFICATION"
	EventUnauthorizedAccess   = "UNAUTHORIZED_ACCESS"
	EventDirectMessageSending = "DIRECT_MESSAGE_SENDING"
	EventWebSocketConnection  = "WEBSOCKET_CONNECTION"
)

// SendAuthErrorResponse sends a standardized authentication error response
func SendAuthErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(fmt.Sprintf(`{"error":true,"message":"%s"}`, message)))
}
