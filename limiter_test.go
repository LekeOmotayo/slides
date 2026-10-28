package main

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindowLimiter(t *testing.T) {
	l := NewSlidingWindowLimiter(150*time.Millisecond, 2)
	client := "client_a"

	res1 := l.AllowWithResult(client)
	if !res1.Allowed || res1.Remaining != 1 {
		t.Fatalf("Expected first request allowed with 1 remaining, got: %+v", res1)
	}

	res2 := l.AllowWithResult(client)
	if !res2.Allowed || res2.Remaining != 0 {
		t.Fatalf("Expected second request allowed with 0 remaining, got: %+v", res2)
	}

	res3 := l.AllowWithResult(client)
	if res3.Allowed {
		t.Fatalf("Expected third request to be rejected")
	}

	time.Sleep(180 * time.Millisecond)

	res4 := l.AllowWithResult(client)
	if !res4.Allowed {
		t.Fatalf("Expected request after window expiry to be allowed, got: %+v", res4)
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

func TestSlidingWindowStaleCleanup(t *testing.T) {
	l := NewSlidingWindowLimiter(100*time.Millisecond, 5)
	l.Allow("temp_client_1")
	l.Allow("temp_client_2")

	time.Sleep(120 * time.Millisecond)

	evicted := l.CleanupStale(100 * time.Millisecond)
	if evicted != 2 {
		t.Errorf("Expected 2 stale clients evicted, got %d", evicted)
	}

	stats := l.GetStats()
	if stats["active_clients"].(int) != 0 {
		t.Errorf("Expected 0 active clients after eviction, got %v", stats["active_clients"])
	}
}

func TestTokenBucketLimiter(t *testing.T) {
	tb := NewTokenBucketLimiter(3.0, 10.0) // 3 tokens, 10 tokens/sec refill

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

func TestTokenBucketFractionalCost(t *testing.T) {
	tb := NewTokenBucketLimiter(10.0, 5.0)

	res1 := tb.AllowNWithResult(6.5)
	if !res1.Allowed || res1.Remaining != 3 {
		t.Errorf("Expected 6.5 tokens deducted with 3 remaining, got: %+v", res1)
	}

	res2 := tb.AllowNWithResult(5.0)
	if res2.Allowed {
		t.Errorf("Expected 5.0 tokens request to be rejected when only 3.5 available")
	}
}

func TestSlidingWindowCounter(t *testing.T) {
	sc := NewSlidingWindowCounter(200*time.Millisecond, 3)
	key := "counter_client"

	if !sc.AllowWithResult(key).Allowed {
		t.Errorf("Req 1 should be allowed")
	}
	if !sc.AllowWithResult(key).Allowed {
		t.Errorf("Req 2 should be allowed")
	}
	if !sc.AllowWithResult(key).Allowed {
		t.Errorf("Req 3 should be allowed")
	}
	if sc.AllowWithResult(key).Allowed {
		t.Errorf("Req 4 should be rejected")
	}
}

func TestMultiTokenBucketLimiter(t *testing.T) {
	mtb := NewMultiTokenBucketLimiter(10.0, 2.0)
	resFree := mtb.AllowN("user_a", "free", 5.0)
	if !resFree.Allowed {
		t.Errorf("Free user with 5 token cap should consume 5 tokens")
	}
	resFree2 := mtb.AllowN("user_a", "free", 1.0)
	if resFree2.Allowed {
		t.Errorf("Free user should now be throttled")
	}

	resPro := mtb.AllowN("user_b", "pro", 25.0)
	if !resPro.Allowed {
		t.Errorf("Pro user with 50 token cap should consume 25 tokens")
	}
}

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	l := NewSlidingWindowLimiter(1*time.Minute, b.N+1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Allow("bench_client")
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucketLimiter(float64(b.N+1000), 1000000.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkSlidingWindowCounter_Allow(b *testing.B) {
	sc := NewSlidingWindowCounter(1*time.Minute, b.N+1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.AllowWithResult("bench_counter")
	}
}
