package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service provides authentication operations.
type Service struct {
	db        *sql.DB
	jwtSecret []byte
	// challenges stores temporary challenges for users.
	challenges sync.Map // map[user_id]challenge string
}

// NewService creates a new authentication service instance.
func NewService(db *sql.DB) *Service {
	// In production, load jwtSecret from secure configuration.
	secret := []byte("your-secret-key")
	return &Service{
		db:        db,
		jwtSecret: secret,
	}
}

// RegistrationPayload is the expected JSON payload for registration.
type RegistrationPayload struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"` // base64-encoded public key
}

func (s *Service) HandleCheckUserID(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user_id from the URL path
	// The URL should be /auth/check-userid/{userid}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	userID := pathParts[3]

	// Check if the user ID exists in the database
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?)", userID).Scan(&exists)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Create and send JSON response
	response := struct {
		Exists bool `json:"exists"`
	}{
		Exists: exists,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// HandleRegistration registers a new user.
func (a *Service) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload RegistrationPayload
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Insert the new user into the database.
	query := `INSERT INTO users (user_id, username, public_key) VALUES (?, ?, ?)`
	_, err = a.db.Exec(query, payload.UserID, payload.Username, payload.PublicKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Registration error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

// HandleGetUserInfo retrieves a user's public key and other information
func (a *Service) HandleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user_id from URL path - format: /auth/users/{user_id}
	urlPath := r.URL.Path
	parts := strings.Split(urlPath, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	userID := parts[len(parts)-1]
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Query database for user info
	query := "SELECT user_id, username, public_key FROM users WHERE user_id = ?"
	var user struct {
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		PublicKey string `json:"public_key"`
	}

	err := a.db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.PublicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return user info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// LoginPayload represents a login request.
type LoginPayload struct {
	UserID string `json:"user_id"`
}

// ChallengeResponsePayload is used to verify the authentication challenge.
type ChallengeResponsePayload struct {
	UserID    string `json:"user_id"`
	Signature string `json:"signature"` // base64-encoded signature
}

// HandleLogin handles both the challenge issuance and the verification phases.
// When the "verify" query parameter is set to "true", the server will verify the challenge response.
func (a *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if this request is to verify the challenge response.
	if r.URL.Query().Get("verify") == "true" {
		a.handleChallengeResponse(w, r)
		return
	}

	var payload LoginPayload
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate a random challenge.
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		http.Error(w, "Failed to generate challenge", http.StatusInternalServerError)
		return
	}
	challenge := base64.StdEncoding.EncodeToString(challengeBytes)
	a.challenges.Store(payload.UserID, challenge)

	// Return the challenge to the client.
	resp := map[string]string{"challenge": challenge}
	jsonResp, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

// handleChallengeResponse verifies the client's signature of the challenge.
func (a *Service) handleChallengeResponse(w http.ResponseWriter, r *http.Request) {
	var payload ChallengeResponsePayload
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Retrieve the challenge that was issued to this user.
	challengeVal, ok := a.challenges.Load(payload.UserID)
	if !ok {
		http.Error(w, "No challenge found", http.StatusBadRequest)
		return
	}
	challenge, ok := challengeVal.(string)
	if !ok {
		http.Error(w, "Invalid challenge", http.StatusInternalServerError)
		return
	}

	// Fetch the user's public key from the database.
	var publicKeyStr string
	query := "SELECT public_key FROM users WHERE user_id = ?"
	err = a.db.QueryRow(query, payload.UserID).Scan(&publicKeyStr)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Decode the public key and signature.
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		http.Error(w, "Invalid public key", http.StatusInternalServerError)
		return
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(payload.Signature)
	if err != nil {
		http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
		return
	}

	// Verify the signature using ed25519.
	if !ed25519.Verify(pubKeyBytes, []byte(challenge), signatureBytes) {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Issue a JWT token valid for 24 hours.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": payload.UserID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"token": tokenString}
	jsonResp, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

// ParseToken validates the JWT token and returns the claims.
func ParseToken(tokenStr string, service *Service) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return service.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
