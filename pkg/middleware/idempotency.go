package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type IdempotencyStore interface {
	Get(key string) (*CachedResponse, bool)
	Set(key string, response *CachedResponse)
	Stop() // Stop cleanup goroutines and release resources
}

type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

type InMemoryIdempotencyStore struct {
	mu     sync.RWMutex
	store  map[string]*CachedResponse
	ttl    time.Duration
	stopCh chan struct{}
}

func NewInMemoryIdempotencyStore(ttl time.Duration) *InMemoryIdempotencyStore {
	store := &InMemoryIdempotencyStore{
		store:  make(map[string]*CachedResponse),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	go store.cleanup()

	return store
}

func (s *InMemoryIdempotencyStore) Get(key string) (*CachedResponse, bool) {
	s.mu.RLock()
	response, exists := s.store[key]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Since(response.CreatedAt) > s.ttl {
		s.mu.Lock()
		delete(s.store, key)
		s.mu.Unlock()
		return nil, false
	}

	return response, true
}

func (s *InMemoryIdempotencyStore) Set(key string, response *CachedResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	response.CreatedAt = time.Now()
	s.store[key] = response
}

func (s *InMemoryIdempotencyStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			for key, response := range s.store {
				if time.Since(response.CreatedAt) > s.ttl {
					delete(s.store, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *InMemoryIdempotencyStore) Stop() {
	close(s.stopCh)
}

type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
	rc.ResponseWriter.WriteHeader(statusCode)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body.Write(b)
	return rc.ResponseWriter.Write(b)
}

func Idempotency(store IdempotencyStore, headerName string) func(http.Handler) http.Handler {
	if headerName == "" {
		headerName = "Idempotency-Key"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idempotencyKey := extractIdempotencyKey(r, headerName)

			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			if handleCachedResponse(w, store, idempotencyKey) {
				return
			}

			capture := captureResponse(w)
			next.ServeHTTP(capture, r)
			cacheSuccessfulResponse(store, idempotencyKey, capture, w)
		})
	}
}

func extractIdempotencyKey(r *http.Request, headerName string) string {
	return r.Header.Get(headerName)
}

func handleCachedResponse(w http.ResponseWriter, store IdempotencyStore, key string) bool {
	cached, found := store.Get(key)
	if !found {
		return false
	}

	replayCachedResponse(w, cached)
	return true
}

func replayCachedResponse(w http.ResponseWriter, cached *CachedResponse) {
	for key, values := range cached.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(cached.StatusCode)
	_, _ = w.Write(cached.Body)
}

func captureResponse(w http.ResponseWriter) *responseCapture {
	return &responseCapture{
		ResponseWriter: w,
		statusCode:     200,
		body:           &bytes.Buffer{},
	}
}

func cacheSuccessfulResponse(store IdempotencyStore, key string, capture *responseCapture, w http.ResponseWriter) {
	if !shouldCacheResponse(capture.statusCode) {
		return
	}

	cached := &CachedResponse{
		StatusCode: capture.statusCode,
		Headers:    w.Header().Clone(),
		Body:       capture.body.Bytes(),
	}
	store.Set(key, cached)
}

func shouldCacheResponse(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
