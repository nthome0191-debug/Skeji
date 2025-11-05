package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// timeoutWriter wraps http.ResponseWriter to prevent writes after timeout
type timeoutWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	timedOut    bool
	written     bool
	statusCode  int
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.written {
		return
	}

	tw.statusCode = code
	tw.written = true
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}

	if !tw.written {
		tw.statusCode = http.StatusOK
		tw.written = true
	}

	return tw.ResponseWriter.Write(b)
}

func (tw *timeoutWriter) timeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.timedOut = true
}

func RequestTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			tw := &timeoutWriter{
				ResponseWriter: w,
				timedOut:       false,
				written:        false,
			}

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(tw, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				tw.timeout()
				tw.mu.Lock()
				if !tw.written {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error":"Request timeout"}`))
					tw.written = true
				}
				tw.mu.Unlock()
			}
		})
	}
}
