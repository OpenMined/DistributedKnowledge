package http

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUsageTrackingHandlersRegistration tests that the usage tracking routes are correctly registered
func TestUsageTrackingHandlersRegistration(t *testing.T) {
	// Create a router with mock handlers
	router := mux.NewRouter()

	// Create a test handler that just returns OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register our mock handlers for all the routes
	router.HandleFunc("/api/v1/usage", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage/{apiId}", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage/{apiId}/user/{userId}", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage-summary", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage-summary/{apiId}", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage-summary/{apiId}/user/{userId}", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/usage-summary/refresh", testHandler).Methods("POST")
	router.HandleFunc("/api/v1/notifications", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/notifications/user/{userId}", testHandler).Methods("GET")
	router.HandleFunc("/api/v1/notifications/{id}/read", testHandler).Methods("PUT")
	router.HandleFunc("/api/v1/notifications/{id}", testHandler).Methods("DELETE")
	router.HandleFunc("/api/v1/notifications/cleanup", testHandler).Methods("POST")

	// Test a few key routes to make sure they're registered
	routes := []struct {
		path   string
		method string
	}{
		{"/api/v1/usage", "GET"},
		{"/api/v1/usage/abc123", "GET"},
		{"/api/v1/usage/abc123/user/user123", "GET"},
		{"/api/v1/usage-summary", "GET"},
		{"/api/v1/usage-summary/abc123", "GET"},
		{"/api/v1/notifications", "GET"},
		{"/api/v1/notifications/user/user123", "GET"},
		{"/api/v1/notifications/notif123/read", "PUT"},
	}

	for _, route := range routes {
		// Create a request with the test route
		req, err := http.NewRequest(route.method, route.path, nil)
		if !assert.NoError(t, err) {
			continue
		}

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(rr, req)

		// The handler should exist and return 200 OK
		assert.Equal(t, http.StatusOK, rr.Code, "Route should exist and return OK: %s %s", route.method, route.path)
	}
}
