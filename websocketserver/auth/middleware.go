package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// AuthenticatedUserID is a context key for storing the authenticated user ID
type AuthenticatedUserID string

// AuthMiddleware provides a reusable authentication middleware
type AuthMiddleware struct {
	authService *Service
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *Service) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth is a middleware that requires authentication for an endpoint
func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify token without requiring specific user ID
		tokenResult := VerifyToken(tokenStr, am.authService, "")
		if !tokenResult.Valid || tokenResult.Error != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", tokenResult.Error), http.StatusUnauthorized)
			return
		}

		// Verify user exists in database
		var userExists bool
		err := am.authService.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?)", tokenResult.UserID).Scan(&userExists)
		if err != nil {
			log.Printf("Database error checking user existence: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !userExists {
			log.Printf("Security alert: Token contains valid signature but non-existent user ID: %s", tokenResult.UserID)
			http.Error(w, "Invalid user", http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthenticatedUserID("user_id"), tokenResult.UserID)

		// Call the next handler with authenticated context
		next(w, r.WithContext(ctx))
	}
}

// RequireSpecificUser ensures that the token belongs to a specific user
func (am *AuthMiddleware) RequireSpecificUser(userID string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify token with specific user ID requirement
		tokenResult := VerifyToken(tokenStr, am.authService, userID)
		if !tokenResult.Valid || tokenResult.Error != nil {
			// Use a generic error message to avoid leaking information
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			// But log the specific issue for debugging
			log.Printf("Authorization failed: %v", tokenResult.Error)
			return
		}

		// Add user ID to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthenticatedUserID("user_id"), tokenResult.UserID)

		// Call the next handler with authenticated context
		next(w, r.WithContext(ctx))
	}
}

// GetAuthenticatedUserID extracts the authenticated user ID from the request context
func GetAuthenticatedUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(AuthenticatedUserID("user_id")).(string)
	return userID, ok
}
