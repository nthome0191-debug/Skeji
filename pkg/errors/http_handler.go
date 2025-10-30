package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		fmt.Fprintf(os.Stdout, "ERROR: Failed to encode error response to JSON: %v (original error: %s)\n", err, appErr.Error())
	}
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return err
	}
	return nil
}
