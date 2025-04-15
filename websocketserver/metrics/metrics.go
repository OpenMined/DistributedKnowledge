// Package metrics provides in-memory counters and aggregators for key engagement metrics.
package metrics

import (
	"fmt"
	"sync"
	"time"
)

// sessionStarts tracks when each session began (using a unique sessionID, e.g. a client pointer string).
var sessionStarts = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

// dailyActiveUsers and weeklyActiveUsers record the latest connection timestamp per user.
var dailyActiveUsers = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

var weeklyActiveUsers = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

// messageCounts aggregates messages per session (keyed by sessionID).
var messageCounts = struct {
	sync.Mutex
	m map[string]*MessageCount
}{m: make(map[string]*MessageCount)}

// MessageCount holds counters for direct and broadcast messages.
type MessageCount struct {
	Direct    int
	Broadcast int
}

// peakUsage tracks the number of session starts per hour.
var peakUsage = struct {
	sync.Mutex
	m map[int]int // key: hour (0-23), value: count
}{m: make(map[int]int)}

// lastSeen tracks when each user last disconnected (for churn analysis).
var lastSeen = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

// sessionDurations aggregates session durations per user.
var sessionDurations = struct {
	sync.Mutex
	m map[string][]time.Duration
}{m: make(map[string][]time.Duration)}

// RecordSessionStart records the start time of a session.
func RecordSessionStart(sessionID, userID string) {
	now := time.Now()
	sessionStarts.Lock()
	sessionStarts.m[sessionID] = now
	sessionStarts.Unlock()

	// Mark this user as active for the day and week.
	dailyActiveUsers.Lock()
	dailyActiveUsers.m[userID] = now
	dailyActiveUsers.Unlock()

	weeklyActiveUsers.Lock()
	weeklyActiveUsers.m[userID] = now
	weeklyActiveUsers.Unlock()

	// Record peak usage hour.
	hour := now.Hour()
	peakUsage.Lock()
	peakUsage.m[hour]++
	peakUsage.Unlock()

	// Initialize message count for this session.
	messageCounts.Lock()
	messageCounts.m[sessionID] = &MessageCount{}
	messageCounts.Unlock()

	fmt.Printf("Metrics: Session %s started for user %s at %s\n", sessionID, userID, now.Format(time.RFC3339))
}

// RecordSessionEnd computes and logs the session duration, then updates the last seen timestamp.
func RecordSessionEnd(sessionID, userID string) {
	now := time.Now()
	sessionStarts.Lock()
	startTime, exists := sessionStarts.m[sessionID]
	if exists {
		duration := now.Sub(startTime)
		sessionDurations.Lock()
		sessionDurations.m[userID] = append(sessionDurations.m[userID], duration)
		sessionDurations.Unlock()
		delete(sessionStarts.m, sessionID)
		fmt.Printf("Metrics: Session %s ended for user %s. Duration: %s\n", sessionID, userID, duration)
	}
	sessionStarts.Unlock()

	// Update last seen for churn analysis.
	lastSeen.Lock()
	lastSeen.m[userID] = now
	lastSeen.Unlock()
}

// RecordMessageSent updates message counters for the session.
func RecordMessageSent(sessionID string, isBroadcast bool) {
	messageCounts.Lock()
	if count, exists := messageCounts.m[sessionID]; exists {
		if isBroadcast {
			count.Broadcast++
		} else {
			count.Direct++
		}
	}
	messageCounts.Unlock()
	fmt.Printf("Metrics: Message sent in session %s. IsBroadcast: %t\n", sessionID, isBroadcast)
}

// GetDailyActiveUsers returns the count of unique users active since the beginning of today.
func GetDailyActiveUsers() int {
	cutoff := time.Now().Truncate(24 * time.Hour)
	count := 0
	dailyActiveUsers.Lock()
	for _, t := range dailyActiveUsers.m {
		if t.After(cutoff) {
			count++
		}
	}
	dailyActiveUsers.Unlock()
	return count
}

// GetWeeklyActiveUsers returns the count of unique users active in the last 7 days.
func GetWeeklyActiveUsers() int {
	cutoff := time.Now().AddDate(0, 0, -7)
	count := 0
	weeklyActiveUsers.Lock()
	for _, t := range weeklyActiveUsers.m {
		if t.After(cutoff) {
			count++
		}
	}
	weeklyActiveUsers.Unlock()
	return count
}

// GetPeakUsageHour returns the hour with the highest recorded session starts.
func GetPeakUsageHour() int {
	peakUsage.Lock()
	defer peakUsage.Unlock()
	maxHour := 0
	maxCount := 0
	for hour, count := range peakUsage.m {
		if count > maxCount {
			maxCount = count
			maxHour = hour
		}
	}
	return maxHour
}

// CalculateChurnRate computes a churn rate over a given period (e.g. 7 days).
// It returns the proportion of users whose last seen timestamp is older than the period.
func CalculateChurnRate(period time.Duration) float64 {
	now := time.Now()
	churned := 0
	total := 0
	lastSeen.Lock()
	for _, lastActive := range lastSeen.m {
		total++
		if now.Sub(lastActive) > period {
			churned++
		}
	}
	lastSeen.Unlock()
	if total == 0 {
		return 0.0
	}
	return float64(churned) / float64(total)
}
