package ws

import (
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		capacity int
	}{
		{"Default Values", 5.0, 10},
		{"High Rate", 100.0, 200},
		{"Low Rate", 0.5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.rate, tt.capacity)
			if rl.rate != tt.rate {
				t.Errorf("Expected rate %f, got %f", tt.rate, rl.rate)
			}
			if rl.capacity != tt.capacity {
				t.Errorf("Expected capacity %d, got %d", tt.capacity, rl.capacity)
			}
			if rl.buckets == nil {
				t.Error("Expected buckets map to be initialized, got nil")
			}
		})
	}
}

func TestRateLimiterAllow(t *testing.T) {
	t.Run("First Request Always Allowed", func(t *testing.T) {
		rl := NewRateLimiter(5.0, 10)
		if !rl.Allow("user1") {
			t.Error("First request should always be allowed")
		}
	})

	t.Run("Burst Handling", func(t *testing.T) {
		rl := NewRateLimiter(5.0, 5)
		userID := "user2"

		// First request should create bucket with full capacity
		if !rl.Allow(userID) {
			t.Error("First request should be allowed")
		}

		// Should be able to make 4 more requests (5 total with capacity of 5)
		for i := 0; i < 4; i++ {
			if !rl.Allow(userID) {
				t.Errorf("Request %d should be allowed within burst capacity", i+2)
			}
		}

		// Next request should be rejected (exceeded capacity)
		if rl.Allow(userID) {
			t.Error("Request should be rejected after exceeding burst capacity")
		}
	})

	t.Run("Rate Refill", func(t *testing.T) {
		rate := 10.0 // 10 tokens per second
		rl := NewRateLimiter(rate, 10)
		userID := "user3"

		// Use all tokens
		for i := 0; i < 10; i++ {
			rl.Allow(userID)
		}

		// Next request should be rejected
		if rl.Allow(userID) {
			t.Error("Request should be rejected after using all tokens")
		}

		// Wait for tokens to refill (0.5 seconds = 5 tokens at rate of 10/sec)
		time.Sleep(500 * time.Millisecond)

		// Should be able to make 5 more requests
		for i := 0; i < 5; i++ {
			if !rl.Allow(userID) {
				t.Errorf("Request %d should be allowed after refill", i+1)
			}
		}

		// Next request should be rejected again
		if rl.Allow(userID) {
			t.Error("Request should be rejected after using refilled tokens")
		}
	})
}

func TestRateLimiterRemoveUser(t *testing.T) {
	rl := NewRateLimiter(5.0, 10)
	userID := "user4"

	// Create bucket for user
	rl.Allow(userID)

	// Verify bucket exists
	rl.lockMap.RLock()
	_, exists := rl.buckets[userID]
	rl.lockMap.RUnlock()
	if !exists {
		t.Error("Expected bucket to exist after Allow() call")
	}

	// Remove user
	rl.RemoveUser(userID)

	// Verify bucket was removed
	rl.lockMap.RLock()
	_, exists = rl.buckets[userID]
	rl.lockMap.RUnlock()
	if exists {
		t.Error("Expected bucket to be removed after RemoveUser() call")
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	rl := NewRateLimiter(10.0, 10)
	userIDs := []string{"user1", "user2", "user3", "user4", "user5"}
	requestsPerUser := 15

	var wg sync.WaitGroup

	// Launch multiple goroutines to simulate concurrent requests
	for _, userID := range userIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			allowedCount := 0
			for i := 0; i < requestsPerUser; i++ {
				if rl.Allow(id) {
					allowedCount++
				}
			}
			// For a new user with capacity 10, expect exactly 10 requests to be allowed
			if allowedCount != 10 {
				t.Errorf("Expected 10 allowed requests for user %s, got %d", id, allowedCount)
			}
		}(userID)
	}

	wg.Wait()

	// Verify all users have buckets
	rl.lockMap.RLock()
	bucketCount := len(rl.buckets)
	rl.lockMap.RUnlock()
	if bucketCount != len(userIDs) {
		t.Errorf("Expected %d buckets, got %d", len(userIDs), bucketCount)
	}
}
