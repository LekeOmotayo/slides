package repositories

import (
	"sync"
	"time"

	"leke-sliding-window-limiter/backend/internal/domain/limiter"
)

// LimiterRepository manages the lifecycle and storage of rate limiter instances.
type LimiterRepository interface {
	GetSlidingWindowLimiter() *limiter.SlidingWindowLimiter
	GetSlidingWindowCounter() *limiter.SlidingWindowCounter
	GetTokenBucketLimiter() *limiter.TokenBucketLimiter
	GetMultiTokenBucketLimiter() *limiter.MultiTokenBucketLimiter
	ResetAll()
}

// InMemLimiterRepository implements LimiterRepository in memory with thread-safety.
type InMemLimiterRepository struct {
	mu             sync.RWMutex
	swLimiter      *limiter.SlidingWindowLimiter
	scLimiter      *limiter.SlidingWindowCounter
	tbLimiter      *limiter.TokenBucketLimiter
	multiTbLimiter *limiter.MultiTokenBucketLimiter
}

// NewInMemLimiterRepository creates an initialized in-memory repository.
func NewInMemLimiterRepository() *InMemLimiterRepository {
	return &InMemLimiterRepository{
		swLimiter:      limiter.NewSlidingWindowLimiter(2*time.Second, 5),
		scLimiter:      limiter.NewSlidingWindowCounter(2*time.Second, 5),
		tbLimiter:      limiter.NewTokenBucketLimiter(10.0, 2.0),
		multiTbLimiter: limiter.NewMultiTokenBucketLimiter(10.0, 2.0),
	}
}

func (r *InMemLimiterRepository) GetSlidingWindowLimiter() *limiter.SlidingWindowLimiter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.swLimiter
}

func (r *InMemLimiterRepository) GetSlidingWindowCounter() *limiter.SlidingWindowCounter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scLimiter
}

func (r *InMemLimiterRepository) GetTokenBucketLimiter() *limiter.TokenBucketLimiter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tbLimiter
}

func (r *InMemLimiterRepository) GetMultiTokenBucketLimiter() *limiter.MultiTokenBucketLimiter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.multiTbLimiter
}

func (r *InMemLimiterRepository) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.swLimiter = limiter.NewSlidingWindowLimiter(2*time.Second, 5)
	r.scLimiter = limiter.NewSlidingWindowCounter(2*time.Second, 5)
	r.tbLimiter = limiter.NewTokenBucketLimiter(10.0, 2.0)
	r.multiTbLimiter = limiter.NewMultiTokenBucketLimiter(10.0, 2.0)
}
