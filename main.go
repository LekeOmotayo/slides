package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("⏱️  Leke Sliding Window & Token Bucket Rate Limiter")
	fmt.Println("============================================================")

	// Sliding window: allow 3 requests per 500 milliseconds
	fmt.Println("\n[1] Testing Sliding Window Limiter (Limit: 3 reqs / 500ms):")
	sw := NewSlidingWindowLimiter(500*time.Millisecond, 3)
	user := "user_tier_1"

	for i := 1; i <= 5; i++ {
		allowed := sw.Allow(user)
		fmt.Printf(" Request #%d -> Allowed: %v\n", i, allowed)
	}

	fmt.Println("\n Sleeping 600ms for window to reset...")
	time.Sleep(600 * time.Millisecond)

	allowedAfterSleep := sw.Allow(user)
	fmt.Printf(" Request after window expiration -> Allowed: %v\n", allowedAfterSleep)

	// Token bucket: 5 capacity, 2 tokens/sec refill
	fmt.Println("\n[2] Testing Token Bucket Limiter (Capacity: 5, Refill: 2/sec):")
	tb := NewTokenBucketLimiter(5.0, 2.0)
	for i := 1; i <= 6; i++ {
		fmt.Printf(" Token Request #%d -> Allowed: %v\n", i, tb.Allow())
	}

	fmt.Println("\n✅ All rate limiting algorithms executed with zero dependencies.")
}
