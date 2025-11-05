package middleware

import (
	"net/http"
	"skeji/pkg/logger"
	"sync"
	"time"
)

type PhoneExtractor func(r *http.Request) string

type PhoneRateLimiter struct {
	mu             sync.RWMutex
	requests       map[string][]time.Time
	limit          int
	window         time.Duration
	phoneExtractor PhoneExtractor
	log            *logger.Logger
	stopCh         chan struct{}
}

func NewPhoneRateLimiter(limit int, window time.Duration, extractor PhoneExtractor, log *logger.Logger) *PhoneRateLimiter {
	limiter := &PhoneRateLimiter{
		requests:       make(map[string][]time.Time),
		limit:          limit,
		window:         window,
		phoneExtractor: extractor,
		log:            log,
		stopCh:         make(chan struct{}),
	}

	go limiter.cleanup()

	return limiter
}

func (rl *PhoneRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for phone, timestamps := range rl.requests {
				if len(timestamps) == 0 || time.Since(timestamps[len(timestamps)-1]) > rl.window {
					delete(rl.requests, phone)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *PhoneRateLimiter) Stop() {
	close(rl.stopCh)
}

func (rl *PhoneRateLimiter) Allow(phone string) bool {
	if phone == "" {
		return true
	}

	now := time.Now()

	rl.mu.RLock()
	timestamps := rl.requests[phone]
	rl.mu.RUnlock()

	validTimestamps := make([]time.Time, 0)
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	if len(validTimestamps) >= rl.limit {
		return false
	}

	validTimestamps = append(validTimestamps, now)

	rl.mu.Lock()
	rl.requests[phone] = validTimestamps
	rl.mu.Unlock()

	return true
}

func PhoneRateLimit(limiter *PhoneRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			phone := extractPhoneNumber(r, limiter.phoneExtractor)

			if phone == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.Allow(phone) {
				rejectRateLimited(w, limiter.log, r, phone)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractPhoneNumber(r *http.Request, extractor PhoneExtractor) string {
	if extractor == nil {
		return r.Header.Get("X-Phone-Number")
	}
	return extractor(r)
}

func rejectRateLimited(w http.ResponseWriter, log *logger.Logger, r *http.Request, phone string) {
	requestID := ""
	if rid := r.Context().Value(RequestIDKey); rid != nil {
		requestID = rid.(string)
	}

	log.Warn("Rate limit exceeded",
		"request_id", requestID,
		"phone", phone,
		"path", r.URL.Path,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"Rate limit exceeded"}`))
}

func DefaultPhoneExtractor(r *http.Request) string {
	return r.Header.Get("X-Phone-Number")
}
