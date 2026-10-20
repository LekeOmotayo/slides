package main

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindowLimiter(t *testing.T) {
	l := NewSlidingWindowLimiter(200*time.Millisecond, 2)
	client := "client_a"

	if !l.Allow(client) {
		t.Errorf("Expected first request to be allowed")
	}
	if !l.Allow(client) {
		t.Errorf("Expected second request to be allowed")
	}
	if l.Allow(client) {
		t.Errorf("Expected third request to be rejected")
	}

	time.Sleep(250 * time.Millisecond)

	if !l.Allow(client) {
		t.Errorf("Expected request after window expiry to be allowed")
	}
}

func TestSlidingWindowConcurrency(t *testing.T) {
	l := NewSlidingWindowLimiter(1*time.Second, 50)
	var wg sync.WaitGroup
	allowedCount := 0
	var countMu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow("concurrent_client") {
				countMu.Lock()
				allowedCount++
				countMu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowedCount != 50 {
		t.Errorf("Expected exactly 50 allowed requests, got %d", allowedCount)
	}
}

func TestTokenBucketLimiter(t *testing.T) {
	tb := NewTokenBucketLimiter(3.0, 10.0)

	if !tb.Allow() || !tb.Allow() || !tb.Allow() {
		t.Errorf("Expected 3 tokens to be available immediately")
	}
	if tb.Allow() {
		t.Errorf("Expected 4th token to be exhausted")
	}

	time.Sleep(150 * time.Millisecond) // Should refill ~1.5 tokens
	if !tb.Allow() {
		t.Errorf("Expected token to be available after refill")
	}
}
