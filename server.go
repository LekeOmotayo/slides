package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Server encapsulates the HTTP state and rate limiter instances.
type Server struct {
	port           int
	swLimiter      *SlidingWindowLimiter
	scLimiter      *SlidingWindowCounter
	tbLimiter      *TokenBucketLimiter
	multiTbLimiter *MultiTokenBucketLimiter
	startTime      time.Time
}

// NewServer initializes a rate limiter web server.
func NewServer(port int) *Server {
	return &Server{
		port:           port,
		swLimiter:      NewSlidingWindowLimiter(2*time.Second, 5), // 5 reqs per 2s
		scLimiter:      NewSlidingWindowCounter(2*time.Second, 5),
		tbLimiter:      NewTokenBucketLimiter(10.0, 2.0), // 10 capacity, 2/sec refill
		multiTbLimiter: NewMultiTokenBucketLimiter(10.0, 2.0),
		startTime:      time.Now(),
	}
}

// RequestPayload represents the incoming API call.
type RequestPayload struct {
	Key       string  `json:"key"`
	Algorithm string  `json:"algorithm"` // sliding_window, sliding_counter, token_bucket
	Cost      int     `json:"cost"`
	Tier      string  `json:"tier"` // free, pro, enterprise
	Tokens    float64 `json:"tokens"`
}

// Start launches the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static assets handler
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	staticDir := filepath.Join(workDir, "static")

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.HandleFunc("/", s.handleIndex)

	// API routes
	mux.HandleFunc("/api/allow", s.handleAllow)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/benchmark", s.handleBenchmark)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🚀 Leke Sliding Window Limiter Server running at http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func (s *Server) handleAllow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		req.Key = "client_default"
	}
	if req.Cost <= 0 {
		req.Cost = 1
	}
	if req.Tier == "" {
		req.Tier = "free"
	}

	var res RateLimiterResult

	switch req.Algorithm {
	case "token_bucket":
		tokensRequested := float64(req.Cost)
		if req.Tokens > 0 {
			tokensRequested = req.Tokens
		}
		res = s.multiTbLimiter.AllowN(req.Key, req.Tier, tokensRequested)
	case "sliding_counter":
		res = s.scLimiter.AllowWithResult(req.Key)
	case "sliding_window":
		fallthrough
	default:
		res = s.swLimiter.AllowNWithResult(req.Key, req.Cost)
	}

	// Attach RFC rate limit headers to HTTP response
	for k, v := range res.HTTPHeaders() {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", "application/json")

	if !res.Allowed {
		w.WriteHeader(http.StatusTooManyRequests) // 429
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(res)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	swStats := s.swLimiter.GetStats()
	payload := map[string]interface{}{
		"sliding_window": swStats,
		"token_bucket": map[string]interface{}{
			"capacity":       s.tbLimiter.capacity,
			"refill_rate":    s.tbLimiter.refillRate,
			"current_tokens": s.tbLimiter.GetTokens(),
			"total_allowed":  s.tbLimiter.totalAllowed,
			"total_rejected": s.tbLimiter.totalRejected,
		},
		"uptime_seconds": time.Since(s.startTime).Seconds(),
		"server_time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.swLimiter.ResetAll()
	s.scLimiter = NewSlidingWindowCounter(2*time.Second, 5)
	s.tbLimiter = NewTokenBucketLimiter(10.0, 2.0)
	s.multiTbLimiter = NewMultiTokenBucketLimiter(10.0, 2.0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset_successful"})
}

// BenchmarkResult stores live execution telemetry.
type BenchmarkResult struct {
	TotalRequests int     `json:"total_requests"`
	Concurrency   int     `json:"concurrency"`
	Allowed       int     `json:"allowed"`
	Rejected      int     `json:"rejected"`
	DurationMs    float64 `json:"duration_ms"`
	OpsPerSec     float64 `json:"ops_per_sec"`
	Algorithm     string  `json:"algorithm"`
}

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Requests    int    `json:"requests"`
		Concurrency int    `json:"concurrency"`
		Algorithm   string `json:"algorithm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Requests = 1000
		req.Concurrency = 25
		req.Algorithm = "sliding_window"
	}
	if req.Requests <= 0 {
		req.Requests = 1000
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 25
	}

	benchLimiter := NewSlidingWindowLimiter(1*time.Second, 500)
	benchCounter := NewSlidingWindowCounter(1*time.Second, 500)
	benchBucket := NewTokenBucketLimiter(500.0, 1000.0)

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	rejected := 0

	reqsPerWorker := req.Requests / req.Concurrency
	for i := 0; i < req.Concurrency; i++ {
		wg.Add(1)
		workerID := fmt.Sprintf("worker_%d", i)
		go func(id string) {
			defer wg.Done()
			localAllowed := 0
			localRejected := 0
			for j := 0; j < reqsPerWorker; j++ {
				var ok bool
				switch req.Algorithm {
				case "token_bucket":
					ok = benchBucket.Allow()
				case "sliding_counter":
					ok = benchCounter.AllowWithResult(id).Allowed
				default:
					ok = benchLimiter.Allow(id)
				}
				if ok {
					localAllowed++
				} else {
					localRejected++
				}
			}
			mu.Lock()
			allowed += localAllowed
			rejected += localRejected
			mu.Unlock()
		}(workerID)
	}
	wg.Wait()
	duration := time.Since(start)

	res := BenchmarkResult{
		TotalRequests: req.Requests,
		Concurrency:   req.Concurrency,
		Allowed:       allowed,
		Rejected:      rejected,
		DurationMs:    float64(duration.Microseconds()) / 1000.0,
		OpsPerSec:     float64(req.Requests) / duration.Seconds(),
		Algorithm:     req.Algorithm,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
