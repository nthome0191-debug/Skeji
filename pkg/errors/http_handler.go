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
		// todo: write out to stdout the failure so it could be tracked
	}
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Return error so caller can log - no recovery possible after WriteHeader
		return err
	}
	return nil
}
