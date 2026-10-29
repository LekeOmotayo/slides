package services

import (
	"errors"
	"time"

	"leke-sliding-window-limiter/backend/internal/api/repositories"
	"leke-sliding-window-limiter/backend/internal/domain/limiter"
)

// RequestPayload represents the incoming rate limit evaluation request.
type RequestPayload struct {
	Key       string  `json:"key"`
	Algorithm string  `json:"algorithm"` // sliding_window, sliding_counter, token_bucket, multi_tier
	Cost      int     `json:"cost"`
	Tier      string  `json:"tier"` // free, pro, enterprise
	Tokens    float64 `json:"tokens"`
}

// LimiterService defines the rate limiting use-cases.
type LimiterService interface {
	Evaluate(payload RequestPayload) (limiter.RateLimiterResult, error)
	GetStats() map[string]interface{}
	Reset()
}

type limiterService struct {
	repo repositories.LimiterRepository
}

// NewLimiterService creates a new LimiterService instance.
func NewLimiterService(repo repositories.LimiterRepository) LimiterService {
	return &limiterService{
		repo: repo,
	}
}

func (s *limiterService) Evaluate(payload RequestPayload) (limiter.RateLimiterResult, error) {
	key := payload.Key
	if key == "" {
		key = "default-client"
	}
	cost := payload.Cost
	if cost <= 0 {
		cost = 1
	}

	switch payload.Algorithm {
	case "sliding_counter":
		return s.repo.GetSlidingWindowCounter().AllowWithResult(key), nil

	case "token_bucket":
		tokens := payload.Tokens
		if tokens <= 0 {
			tokens = float64(cost)
		}
		return s.repo.GetTokenBucketLimiter().AllowNWithResult(tokens), nil

	case "multi_tier":
		tier := payload.Tier
		if tier == "" {
			tier = "free"
		}
		tokens := payload.Tokens
		if tokens <= 0 {
			tokens = float64(cost)
		}
		return s.repo.GetMultiTokenBucketLimiter().AllowN(key, tier, tokens), nil

	case "sliding_window", "":
		return s.repo.GetSlidingWindowLimiter().AllowNWithResult(key, cost), nil

	default:
		return limiter.RateLimiterResult{}, errors.New("unsupported algorithm: " + payload.Algorithm)
	}
}

func (s *limiterService) GetStats() map[string]interface{} {
	swStats := s.repo.GetSlidingWindowLimiter().GetStats()

	return map[string]interface{}{
		"sliding_window": swStats,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *limiterService) Reset() {
	s.repo.ResetAll()
}
