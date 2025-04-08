package ws

import (
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality for WebSocket connections.
type RateLimiter struct {
	// lockMap protects access to the buckets map
	lockMap sync.RWMutex
	// buckets stores the rate limit state for each user
	buckets map[string]*TokenBucket
	// rate is tokens per second
	rate float64
	// capacity is the maximum number of tokens in a bucket
	capacity int
}

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	// mu protects all bucket fields from concurrent access
	mu sync.Mutex
	// tokens is the current number of tokens in the bucket
	tokens float64
	// lastRefill is the last time the bucket was refilled
	lastRefill time.Time
	// capacity is the maximum number of tokens in the bucket
	capacity int
	// rate is tokens per second
	rate float64
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

// Allow returns true if the request is allowed for the given user.
func (rl *RateLimiter) Allow(userID string) bool {
	// First, try to get an existing bucket with a read lock
	rl.lockMap.RLock()
	bucket, exists := rl.buckets[userID]
	rl.lockMap.RUnlock()

	if !exists {
		// If the bucket doesn't exist, acquire a write lock to create it
		rl.lockMap.Lock()
		// Check again after acquiring the write lock (double-checked locking)
		bucket, exists = rl.buckets[userID]
		if !exists {
			// Create a new bucket for this user, starting with 1 less than capacity
			// to account for this first request
			bucket = &TokenBucket{
				tokens:     float64(rl.capacity) - 1.0, // Consume one token for the first request
				lastRefill: time.Now(),
				capacity:   rl.capacity,
				rate:       rl.rate,
			}
			rl.buckets[userID] = bucket
			rl.lockMap.Unlock()
			return true
		}
		rl.lockMap.Unlock()
	}

	// For existing buckets, we need to lock the bucket itself
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.lastRefill = now

	// Calculate how many tokens to add based on the elapsed time and rate
	newTokens := elapsed * bucket.rate
	bucket.tokens = min(float64(bucket.capacity), bucket.tokens+newTokens)

	// Check if we have at least one token and consume it
	if bucket.tokens >= 1.0 {
		bucket.tokens--
		return true
	}

	return false
}

// RemoveUser removes a user from the rate limiter.
func (rl *RateLimiter) RemoveUser(userID string) {
	rl.lockMap.Lock()
	delete(rl.buckets, userID)
	rl.lockMap.Unlock()
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
