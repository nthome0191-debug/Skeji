package middleware

import (
	"net/http"
	"skeji/pkg/logger"
	"strings"
)

func ContentTypeValidation(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requiresContentType(r.Method) {
				contentType := extractContentType(r.Header.Get("Content-Type"))

				if contentType != "application/json" {
					rejectInvalidContentType(w, log, r, contentType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requiresContentType(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
}

func extractContentType(header string) string {
	if header == "" {
		return ""
	}

	parts := strings.Split(header, ";")
	return strings.TrimSpace(parts[0])
}

func rejectInvalidContentType(w http.ResponseWriter, log *logger.Logger, r *http.Request, contentType string) {
	requestID := ""
	if rid := r.Context().Value(RequestIDKey); rid != nil {
		if id, ok := rid.(string); ok {
			requestID = id
		}
	}

	log.Warn("Invalid Content-Type header",
		"request_id", requestID,
		"content_type", contentType,
		"path", r.URL.Path,
		"method", r.Method,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnsupportedMediaType)
	_, _ = w.Write([]byte(`{"error":"Content-Type must be application/json"}`))
}
