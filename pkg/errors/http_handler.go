package errors

import (
	"encoding/json"
	"net/http"
)

// WriteError writes an AppError as a JSON response
// This is a helper for HTTP handlers to consistently return errors
func WriteError(w http.ResponseWriter, err error) {
	// Convert to AppError if not already
	appErr := AsAppError(err)

	// Set status code
	w.WriteHeader(appErr.StatusCode())

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write JSON response
	response := ErrorResponse{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteSuccess writes a successful JSON response
// This is a helper for HTTP handlers to consistently return success responses
func WriteSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
