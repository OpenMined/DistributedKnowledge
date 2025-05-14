package http

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"dk/db"
)

// UsageMetrics represents metrics collected for an API request
type UsageMetrics struct {
	APIID           string
	ExternalUserID  string
	RequestCount    int
	TokensUsed      int
	CreditsConsumed float64
	ExecutionTimeMs int
	Endpoint        string
	WasThrottled    bool
	WasBlocked      bool
}

// PolicyEnforcementMiddleware creates middleware for tracking usage and enforcing policies
func PolicyEnforcementMiddleware(dbConn *db.DatabaseConnection) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to API endpoints
			if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}

			// Get API ID from request (typically from authentication or URL)
			// For this implementation, we'll extract it from a header
			apiID := r.Header.Get("X-API-ID")
			if apiID == "" {
				// If no API ID, just pass through without tracking
				next.ServeHTTP(w, r)
				return
			}

			// Get user ID from request (typically from authentication)
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				// If no user ID, just pass through without tracking
				next.ServeHTTP(w, r)
				return
			}

			// 1. Check if the user has access to this API
			access, err := db.GetAPIUserAccessByUserID(dbConn.DB, apiID, userID)
			if err != nil || !access.IsActive {
				// User doesn't have access, return 403
				http.Error(w, "Access denied: User does not have permission to use this API", http.StatusForbidden)
				return
			}

			// 2. Get the API to determine its policy
			api, err := db.GetAPI(dbConn.DB, apiID)
			if err != nil {
				http.Error(w, "API not found", http.StatusNotFound)
				return
			}

			if !api.IsActive {
				http.Error(w, "API is inactive", http.StatusForbidden)
				return
			}

			// Skip policy check if no policy is assigned or it's a free policy
			var shouldEnforcePolicy bool
			var policy *db.Policy

			if api.PolicyID != nil {
				// Get policy with rules
				policy, err = db.GetPolicyWithRules(dbConn.DB, *api.PolicyID)
				if err != nil {
					// Log error but continue - default to allowing the request
					fmt.Printf("Error getting policy: %v\n", err)
				} else {
					shouldEnforcePolicy = policy.IsActive && policy.Type != "free"
				}
			}

			// Create a response wrapper to capture metrics
			rw := newResponseWriter(w)
			startTime := time.Now()

			// 3. Check policy rules before processing
			if shouldEnforcePolicy {
				// Get current usage summaries
				now := time.Now()
				// For simplicity, we'll just check daily usage
				startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

				usage, err := db.GetTotalUsageForPeriod(dbConn.DB, apiID, userID, startOfDay, endOfDay)
				if err != nil {
					// Log error but continue - assume no usage if we can't get it
					fmt.Printf("Error getting usage: %v\n", err)
				}

				// Check policy rules
				for _, rule := range policy.Rules {
					switch rule.Action {
					case "block":
						// Check if limit is exceeded
						if isLimitExceeded(rule, usage) {
							// Record blocked request
							recordBlockedRequest(dbConn.DB, apiID, userID, r.URL.Path)

							// Create notification
							createQuotaNotification(dbConn.DB, apiID, userID, rule, 100.0, "limit_reached")

							// Return 429 status code
							http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
							return
						}
					case "throttle":
						// Check if throttling is needed
						if isLimitExceeded(rule, usage) {
							// Apply artificial delay
							time.Sleep(500 * time.Millisecond)

							// Record that we throttled
							recordThrottledRequest(dbConn.DB, apiID, userID, r.URL.Path)

							// Create notification
							createQuotaNotification(dbConn.DB, apiID, userID, rule, 100.0, "limit_reached")
						}
					case "notify":
						// Check if notification threshold is reached (80%)
						if isApproachingLimit(rule, usage) {
							// Create notification
							createQuotaNotification(dbConn.DB, apiID, userID, rule, 80.0, "approaching_limit")
						}
					}
				}
			}

			// 4. Serve the request
			next.ServeHTTP(rw, r)

			// 5. Calculate metrics
			duration := time.Since(startTime)

			// Estimate token usage (this would be more accurate if we had actual token count)
			// For this implementation, we'll use a simple heuristic
			responseSize := rw.size
			estimatedTokens := responseSize / 4 // Rough estimate: 4 bytes per token

			// Create usage metrics
			metrics := &UsageMetrics{
				APIID:           apiID,
				ExternalUserID:  userID,
				RequestCount:    1,
				TokensUsed:      estimatedTokens,
				CreditsConsumed: float64(estimatedTokens) * 0.001, // Example credit calculation
				ExecutionTimeMs: int(duration.Milliseconds()),
				Endpoint:        r.URL.Path,
				WasThrottled:    rw.isThrottled,
				WasBlocked:      false, // Not blocked since we're executing this code
			}

			// 6. Record usage
			go recordUsage(dbConn, metrics)
		})
	}
}

// responseWriter is a custom ResponseWriter that tracks response size
type responseWriter struct {
	http.ResponseWriter
	size        int
	isThrottled bool
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

// Write implements the http.ResponseWriter interface
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// isLimitExceeded checks if a rule's limit is exceeded by current usage
func isLimitExceeded(rule db.PolicyRule, usage *db.APIUsageSummary) bool {
	if usage == nil {
		return false
	}

	switch rule.RuleType {
	case "token":
		return float64(usage.TotalTokens) >= rule.LimitValue
	case "request":
		return float64(usage.TotalRequests) >= rule.LimitValue
	case "credit":
		return usage.TotalCredits >= rule.LimitValue
	case "time":
		return float64(usage.TotalTimeMs) >= rule.LimitValue*1000 // Convert to ms
	default:
		return false
	}
}

// isApproachingLimit checks if usage is approaching a rule's limit (80%)
func isApproachingLimit(rule db.PolicyRule, usage *db.APIUsageSummary) bool {
	if usage == nil {
		return false
	}

	threshold := rule.LimitValue * 0.8 // 80% of limit

	switch rule.RuleType {
	case "token":
		return float64(usage.TotalTokens) >= threshold
	case "request":
		return float64(usage.TotalRequests) >= threshold
	case "credit":
		return usage.TotalCredits >= threshold
	case "time":
		return float64(usage.TotalTimeMs) >= threshold*1000 // Convert to ms
	default:
		return false
	}
}

// recordBlockedRequest records a blocked request
func recordBlockedRequest(dbConn *sql.DB, apiID, userID, endpoint string) {
	usage := &db.APIUsage{
		ID:              uuid.New().String(),
		APIID:           apiID,
		ExternalUserID:  userID,
		Timestamp:       time.Now(),
		RequestCount:    1,
		TokensUsed:      0,
		CreditsConsumed: 0,
		ExecutionTimeMs: 0,
		Endpoint:        endpoint,
		WasThrottled:    false,
		WasBlocked:      true,
	}

	err := db.RecordAPIUsage(dbConn, usage)
	if err != nil {
		fmt.Printf("Error recording blocked request: %v\n", err)
	}
}

// recordThrottledRequest marks a request as throttled
func recordThrottledRequest(dbConn *sql.DB, apiID, userID, endpoint string) {
	// Nothing to do here since we can't set a flag on *sql.DB
	// The WasThrottled flag should be set in the responseWriter
}

// recordUsage records API usage metrics
func recordUsage(dbConn *db.DatabaseConnection, metrics *UsageMetrics) {
	usage := &db.APIUsage{
		ID:              uuid.New().String(),
		APIID:           metrics.APIID,
		ExternalUserID:  metrics.ExternalUserID,
		Timestamp:       time.Now(),
		RequestCount:    metrics.RequestCount,
		TokensUsed:      metrics.TokensUsed,
		CreditsConsumed: metrics.CreditsConsumed,
		ExecutionTimeMs: metrics.ExecutionTimeMs,
		Endpoint:        metrics.Endpoint,
		WasThrottled:    metrics.WasThrottled,
		WasBlocked:      metrics.WasBlocked,
	}

	// Record raw usage
	err := db.RecordAPIUsage(dbConn.DB, usage)
	if err != nil {
		fmt.Printf("Error recording API usage: %v\n", err)
		return
	}

	// Update daily summary
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

	// Get current summary
	currentUsage, err := db.GetTotalUsageForPeriod(dbConn.DB, metrics.APIID, metrics.ExternalUserID, startOfDay, endOfDay)
	if err != nil {
		fmt.Printf("Error getting current usage: %v\n", err)
		return
	}

	// Prepare summary
	summary := &db.APIUsageSummary{
		APIID:             metrics.APIID,
		ExternalUserID:    metrics.ExternalUserID,
		PeriodType:        "daily",
		PeriodStart:       startOfDay,
		PeriodEnd:         endOfDay,
		TotalRequests:     currentUsage.TotalRequests,
		TotalTokens:       currentUsage.TotalTokens,
		TotalCredits:      currentUsage.TotalCredits,
		TotalTimeMs:       currentUsage.TotalTimeMs,
		ThrottledRequests: currentUsage.ThrottledRequests,
		BlockedRequests:   currentUsage.BlockedRequests,
		LastUpdated:       time.Now(),
	}

	// Update summary
	err = db.UpsertAPIUsageSummary(dbConn.DB, summary)
	if err != nil {
		fmt.Printf("Error updating usage summary: %v\n", err)
	}
}

// createQuotaNotification creates a quota notification
func createQuotaNotification(dbConn *sql.DB, apiID, userID string, rule db.PolicyRule, percentageUsed float64, notificationType string) {
	// Check if a similar notification was recently created
	// In a real implementation, you'd want to avoid duplicate notifications

	notification := &db.QuotaNotification{
		ID:               uuid.New().String(),
		APIID:            apiID,
		ExternalUserID:   userID,
		NotificationType: notificationType,
		RuleType:         rule.RuleType,
		PercentageUsed:   percentageUsed,
		Message:          fmt.Sprintf("%s limit for %s is %0.1f%% used", rule.RuleType, rule.Period, percentageUsed),
		CreatedAt:        time.Now(),
		IsRead:           false,
	}

	err := db.CreateQuotaNotification(dbConn, notification)
	if err != nil {
		fmt.Printf("Error creating quota notification: %v\n", err)
	}
}
