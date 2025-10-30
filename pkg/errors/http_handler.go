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

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		// todo: logger.error
	}
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		// todo: logger.error
	}
}
