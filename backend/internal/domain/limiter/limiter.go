package limiter

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiterResult contains comprehensive telemetry returned after every rate limit evaluation.
// This directly maps to standard IETF RateLimit headers (RFC 6585, draft-ietf-httpapi-ratelimit-headers).
type RateLimiterResult struct {
	Allowed       bool          `json:"allowed"`
	Algorithm     string        `json:"algorithm"`
	Key           string        `json:"key"`
	Limit         int           `json:"limit"`
	Remaining     int           `json:"remaining"`
	ResetAfter    time.Duration `json:"reset_after_ns"`
	ResetAfterSec float64       `json:"reset_after_sec"`
	CurrentUsage  int           `json:"current_usage"`
	Cost          int           `json:"cost"`
	Timestamp     time.Time     `json:"timestamp"`
}

// HTTPHeaders returns standard RFC-compliant rate limit response headers.
func (r RateLimiterResult) HTTPHeaders() map[string]string {
	headers := map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", r.Limit),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", r.Remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%.2f", r.ResetAfterSec),
		"X-RateLimit-Algorithm": r.Algorithm,
	}
	if !r.Allowed {
		headers["Retry-After"] = fmt.Sprintf("%.0f", math.Ceil(r.ResetAfterSec))
	}
	return headers
}

// ============================================================================
// 1. Sliding Window Log (Exact Precision - O(N) memory per active window)
// ============================================================================

// SlidingWindowLimiter implements a high-throughput, thread-safe in-memory sliding window log.
// It stores timestamps of all requests in the active window, offering exact sub-millisecond precision.
type SlidingWindowLimiter struct {
	mu            sync.RWMutex
	windowSize    time.Duration
	maxRequests   int
	clients       map[string][]time.Time
	lastActivity  map[string]time.Time
	totalAllowed  int64
	totalRejected int64
}

// NewSlidingWindowLimiter creates an exact sliding window log limiter.
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:   windowSize,
		maxRequests:  maxRequests,
		clients:      make(map[string][]time.Time),
		lastActivity: make(map[string]time.Time),
	}
}

// Allow checks if a single request from key is permitted.
func (l *SlidingWindowLimiter) Allow(key string) bool {
	res := l.AllowNWithResult(key, 1)
	return res.Allowed
}

// AllowWithResult checks 1 request and returns full telemetry and header info.
func (l *SlidingWindowLimiter) AllowWithResult(key string) RateLimiterResult {
	return l.AllowNWithResult(key, 1)
}

// AllowNWithResult checks cost units against the sliding window.
func (l *SlidingWindowLimiter) AllowNWithResult(key string, cost int) RateLimiterResult {
	if cost <= 0 {
		cost = 1
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-l.windowSize)

	timestamps := l.clients[key]

	// Prune timestamps older than active sliding window boundary
	validIdx := 0
	for i, ts := range timestamps {
		if ts.After(threshold) {
			validIdx = i
			break
		}
		if i == len(timestamps)-1 {
			validIdx = len(timestamps)
		}
	}
	timestamps = timestamps[validIdx:]

	currentUsage := len(timestamps)
	allowed := (currentUsage + cost) <= l.maxRequests

	var resetAfter time.Duration
	if len(timestamps) > 0 {
		oldest := timestamps[0]
		timeSinceOldest := now.Sub(oldest)
		if timeSinceOldest < l.windowSize {
			resetAfter = l.windowSize - timeSinceOldest
		}
	}

	res := RateLimiterResult{
		Allowed:       allowed,
		Algorithm:     "Sliding Window Log",
		Key:           key,
		Limit:         l.maxRequests,
		Cost:          cost,
		Timestamp:     now,
		ResetAfter:    resetAfter,
		ResetAfterSec: resetAfter.Seconds(),
	}

	if allowed {
		for i := 0; i < cost; i++ {
			timestamps = append(timestamps, now)
		}
		l.clients[key] = timestamps
		l.lastActivity[key] = now
		res.CurrentUsage = len(timestamps)
		res.Remaining = l.maxRequests - len(timestamps)
		atomic.AddInt64(&l.totalAllowed, int64(cost))
	} else {
		l.clients[key] = timestamps
		res.CurrentUsage = currentUsage
		res.Remaining = 0
		atomic.AddInt64(&l.totalRejected, 1)
	}

	return res
}

// CleanupStale evicts client keys that haven't made requests in idleTimeout.
// Essential for preventing memory leaks in high-cardinality environments.
func (l *SlidingWindowLimiter) CleanupStale(idleTimeout time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	evicted := 0
	for key, lastSeen := range l.lastActivity {
		if now.Sub(lastSeen) > idleTimeout {
			delete(l.clients, key)
			delete(l.lastActivity, key)
			evicted++
		}
	}
	return evicted
}

// Reset clears state for a specific key.
func (l *SlidingWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.clients, key)
	delete(l.lastActivity, key)
}

// ResetAll purges all client windows and counters.
func (l *SlidingWindowLimiter) ResetAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.clients = make(map[string][]time.Time)
	l.lastActivity = make(map[string]time.Time)
	atomic.StoreInt64(&l.totalAllowed, 0)
	atomic.StoreInt64(&l.totalRejected, 0)
}

// GetStats returns summary telemetry.
func (l *SlidingWindowLimiter) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	totalEntries := 0
	for _, ts := range l.clients {
		totalEntries += len(ts)
	}

	return map[string]interface{}{
		"active_clients":  len(l.clients),
		"stored_entries":  totalEntries,
		"window_size_ms":  l.windowSize.Milliseconds(),
		"max_requests":    l.maxRequests,
		"total_allowed":   atomic.LoadInt64(&l.totalAllowed),
		"total_rejected":  atomic.LoadInt64(&l.totalRejected),
		"algorithm_class": "Exact Sliding Window Log (O(N) memory per client window)",
	}
}

// ============================================================================
// 2. Sliding Window Counter (Cloudflare / Redis Hybrid - O(1) Memory)
// ============================================================================

// windowBucket stores discrete request counters for a slice of time.
type windowBucket struct {
	windowStart time.Time
	count       int
}

// SlidingWindowCounter implements the Cloudflare / Redis hybrid sliding window.
// It approximates rate with O(1) memory by calculating a weighted sum of the
// previous window and current window:
// Count = prevCount * (1 - fractionElapsed) + currentCount.
type SlidingWindowCounter struct {
	mu            sync.RWMutex
	windowSize    time.Duration
	maxRequests   int
	current       map[string]*windowBucket
	previous      map[string]*windowBucket
	totalAllowed  int64
	totalRejected int64
}

// NewSlidingWindowCounter creates a memory-bounded O(1) sliding window counter.
func NewSlidingWindowCounter(windowSize time.Duration, maxRequests int) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		current:     make(map[string]*windowBucket),
		previous:    make(map[string]*windowBucket),
	}
}

// AllowWithResult evaluates the sliding counter with weighted historical approximation.
func (sc *SlidingWindowCounter) AllowWithResult(key string) RateLimiterResult {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	now := time.Now()
	currBucket, exists := sc.current[key]

	if !exists || now.Sub(currBucket.windowStart) >= sc.windowSize {
		// Window rotated: promote current to previous, start fresh current
		if exists && now.Sub(currBucket.windowStart) < 2*sc.windowSize {
			sc.previous[key] = currBucket
		} else {
			delete(sc.previous, key)
		}
		currBucket = &windowBucket{
			windowStart: now.Truncate(sc.windowSize),
			count:       0,
		}
		sc.current[key] = currBucket
	}

	// Calculate weighted estimate from previous bucket
	prevCount := 0
	if prevBucket, ok := sc.previous[key]; ok {
		prevCount = prevBucket.count
	}

	elapsed := now.Sub(currBucket.windowStart)
	weight := 1.0 - (float64(elapsed) / float64(sc.windowSize))
	if weight < 0 {
		weight = 0
	}

	estimatedCount := int(math.Round(float64(prevCount)*weight)) + currBucket.count
	allowed := (estimatedCount + 1) <= sc.maxRequests

	remainingTime := sc.windowSize - elapsed
	res := RateLimiterResult{
		Allowed:       allowed,
		Algorithm:     "Sliding Window Counter (Cloudflare O(1) Hybrid)",
		Key:           key,
		Limit:         sc.maxRequests,
		Cost:          1,
		Timestamp:     now,
		ResetAfter:    remainingTime,
		ResetAfterSec: remainingTime.Seconds(),
	}

	if allowed {
		currBucket.count++
		res.CurrentUsage = estimatedCount + 1
		res.Remaining = sc.maxRequests - (estimatedCount + 1)
		atomic.AddInt64(&sc.totalAllowed, 1)
	} else {
		res.CurrentUsage = estimatedCount
		res.Remaining = 0
		atomic.AddInt64(&sc.totalRejected, 1)
	}

	return res
}

// ============================================================================
// 3. Token Bucket Limiter (Sub-millisecond Refill & Burst Capacity)
// ============================================================================

// TokenBucketLimiter implements a thread-safe token bucket rate limiter.
// Ideal for bursting APIs, network traffic shaping, and LLM token budget control.
type TokenBucketLimiter struct {
	mu            sync.Mutex
	capacity      float64
	tokens        float64
	refillRate    float64 // tokens per second
	lastRefilled  time.Time
	totalAllowed  int64
	totalRejected int64
}

// NewTokenBucketLimiter creates a token bucket with capacity and refill rate (tokens/sec).
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
	res := tb.AllowNWithResult(tokensRequested)
	return res.Allowed
}

// AllowNWithResult consumes tokens and returns full rate limit telemetry.
func (tb *TokenBucketLimiter) AllowNWithResult(tokensRequested float64) RateLimiterResult {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefilled).Seconds()
	tb.lastRefilled = now

	// Refill tokens proportionally to elapsed time
	tb.tokens = math.Min(tb.capacity, tb.tokens+(elapsed*tb.refillRate))

	allowed := tb.tokens >= tokensRequested

	var resetAfter time.Duration
	if !allowed && tb.refillRate > 0 {
		needed := tokensRequested - tb.tokens
		resetAfter = time.Duration((needed / tb.refillRate) * float64(time.Second))
	}

	res := RateLimiterResult{
		Allowed:       allowed,
		Algorithm:     "Token Bucket (Leaky Refill)",
		Key:           "global_bucket",
		Limit:         int(tb.capacity),
		Cost:          int(math.Ceil(tokensRequested)),
		Timestamp:     now,
		ResetAfter:    resetAfter,
		ResetAfterSec: resetAfter.Seconds(),
	}

	if allowed {
		tb.tokens -= tokensRequested
		res.Remaining = int(math.Floor(tb.tokens))
		res.CurrentUsage = int(tb.capacity) - res.Remaining
		atomic.AddInt64(&tb.totalAllowed, int64(math.Ceil(tokensRequested)))
	} else {
		res.Remaining = int(math.Floor(tb.tokens))
		res.CurrentUsage = int(tb.capacity) - res.Remaining
		atomic.AddInt64(&tb.totalRejected, 1)
	}

	return res
}

// GetTokens returns current available tokens.
func (tb *TokenBucketLimiter) GetTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastRefilled).Seconds()
	return math.Min(tb.capacity, tb.tokens+(elapsed*tb.refillRate))
}

// ============================================================================
// 4. Multi-Tenant Keyed Token Bucket Manager
// ============================================================================

// MultiTokenBucketLimiter manages per-client token buckets with tier-based configs.
type MultiTokenBucketLimiter struct {
	mu           sync.RWMutex
	buckets      map[string]*TokenBucketLimiter
	defaultCap   float64
	defaultRate  float64
	tierProfiles map[string]struct {
		Capacity float64
		Refill   float64
	}
}

// NewMultiTokenBucketLimiter creates a multi-client bucket manager.
func NewMultiTokenBucketLimiter(defaultCap, defaultRate float64) *MultiTokenBucketLimiter {
	m := &MultiTokenBucketLimiter{
		buckets:      make(map[string]*TokenBucketLimiter),
		defaultCap:   defaultCap,
		defaultRate:  defaultRate,
		tierProfiles: make(map[string]struct{ Capacity, Refill float64 }),
	}
	m.RegisterTier("free", 5.0, 1.0)
	m.RegisterTier("pro", 50.0, 10.0)
	m.RegisterTier("enterprise", 500.0, 100.0)
	return m
}

// RegisterTier defines rate limit profiles for client tiers.
func (m *MultiTokenBucketLimiter) RegisterTier(tier string, capacity, refillRate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tierProfiles[tier] = struct{ Capacity, Refill float64 }{capacity, refillRate}
}

// GetOrCreate retrieves or creates a token bucket for key.
func (m *MultiTokenBucketLimiter) GetOrCreate(key, tier string) *TokenBucketLimiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.buckets[key]; ok {
		return b
	}

	cap := m.defaultCap
	rate := m.defaultRate
	if prof, ok := m.tierProfiles[tier]; ok {
		cap = prof.Capacity
		rate = prof.Refill
	}

	bucket := NewTokenBucketLimiter(cap, rate)
	m.buckets[key] = bucket
	return bucket
}

// AllowN evaluates request against client's token bucket.
func (m *MultiTokenBucketLimiter) AllowN(key, tier string, cost float64) RateLimiterResult {
	bucket := m.GetOrCreate(key, tier)
	res := bucket.AllowNWithResult(cost)
	res.Key = key
	return res
}

