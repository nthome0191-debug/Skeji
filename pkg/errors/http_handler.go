package errors

import (
	"encoding/json"
	"net/http"
)

func WriteError(w http.ResponseWriter, err error) {
	appErr := AsAppError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode())

	response := ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback: write a plain text error if JSON encoding fails
		http.Error(w, "Internal server error: failed to encode error response", http.StatusInternalServerError)
	}
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Note: We can't change status code here as WriteHeader was already called.
		// The response body may be partially written. This is a critical error
		// that should be logged by the handler layer.
		// For now, we silently fail as there's no recovery possible at this point.
	}
}
