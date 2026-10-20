package main

import (
	"sync"
	"time"
)

// SlidingWindowLimiter implements a high-throughput, thread-safe in-memory sliding window rate limiter.
type SlidingWindowLimiter struct {
	mu          sync.Mutex
	windowSize  time.Duration
	maxRequests int
	clients     map[string][]time.Time
}

// NewSlidingWindowLimiter creates a rate limiter for a specific window duration and limit.
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		clients:     make(map[string][]time.Time),
	}
}

// Allow checks if a request from key is permitted under the sliding window.
func (l *SlidingWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-l.windowSize)

	timestamps := l.clients[key]

	// Prune timestamps older than the active window
	validIndex := 0
	for i, ts := range timestamps {
		if ts.After(threshold) {
			validIndex = i
			break
		}
		if i == len(timestamps)-1 {
			validIndex = len(timestamps)
		}
	}
	timestamps = timestamps[validIndex:]

	// Check capacity
	if len(timestamps) < l.maxRequests {
		timestamps = append(timestamps, now)
		l.clients[key] = timestamps
		return true
	}

	l.clients[key] = timestamps
	return false
}

// TokenBucketLimiter implements a thread-safe token bucket rate limiter.
type TokenBucketLimiter struct {
	mu           sync.Mutex
	capacity     float64
	tokens       float64
	refillRate   float64 // tokens per second
	lastRefilled time.Time
}

// NewTokenBucketLimiter creates a token bucket with max capacity and refill rate.
func NewTokenBucketLimiter(capacity float64, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:     capacity,
		tokens:       capacity,
		refillRate:   refillRate,
		lastRefilled: time.Now(),
	}
}

// Allow consumes 1 token if available.
func (tb *TokenBucketLimiter) Allow() bool {
	return tb.AllowN(1.0)
}

// AllowN consumes n tokens if available.
func (tb *TokenBucketLimiter) AllowN(tokensRequested float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefilled).Seconds()
	tb.lastRefilled = now

	// Refill tokens based on elapsed time
	tb.tokens = tb.tokens + (elapsed * tb.refillRate)
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tb.tokens >= tokensRequested {
		tb.tokens -= tokensRequested
		return true
	}

	return false
}
