package middleware

import (
	"net/http"
	"runtime/debug"
	"skeji/pkg/logger"
)

func Recovery(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := ""
					if rid := r.Context().Value(RequestIDKey); rid != nil {
						if id, ok := rid.(string); ok {
							requestID = id
						}
					}

					log.Error("Panic recovered",
						"request_id", requestID,
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
