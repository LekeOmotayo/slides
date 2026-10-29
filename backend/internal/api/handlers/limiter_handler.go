package handlers

import (
	"encoding/json"
	"net/http"

	"leke-sliding-window-limiter/backend/internal/api/services"
)

// LimiterHandler provides HTTP endpoints for rate limiting operations.
type LimiterHandler struct {
	service services.LimiterService
}

// NewLimiterHandler creates an initialized HTTP handler.
func NewLimiterHandler(service services.LimiterService) *LimiterHandler {
	return &LimiterHandler{
		service: service,
	}
}

// HandleAllow evaluates rate limits for incoming requests.
func (h *LimiterHandler) HandleAllow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload services.RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// Default to empty payload if body is empty or invalid
		payload = services.RequestPayload{
			Key:       "default-client",
			Algorithm: "sliding_window",
			Cost:      1,
		}
	}

	result, err := h.service.Evaluate(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Apply standard IETF RFC RateLimit headers
	for k, v := range result.HTTPHeaders() {
		w.Header().Set(k, v)
	}

	w.Header().Set("Content-Type", "application/json")
	if !result.Allowed {
		w.WriteHeader(http.StatusTooManyRequests)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(result)
}

// HandleStats returns metrics across all rate limiting algorithms.
func (h *LimiterHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.service.GetStats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// HandleReset resets counters for demonstration purposes.
func (h *LimiterHandler) HandleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.service.Reset()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "all limiter states cleared"})
}

// HandleHealth provides a health check endpoint.
func (h *LimiterHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "leke-sliding-window-limiter",
	})
}
