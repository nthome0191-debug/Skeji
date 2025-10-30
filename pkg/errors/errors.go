package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Error codes for consistent error handling across microservices
const (
	CodeNotFound      = "NOT_FOUND"
	CodeValidation    = "VALIDATION_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeConflict      = "CONFLICT"
	CodeInternal      = "INTERNAL_ERROR"
	CodeBadRequest    = "BAD_REQUEST"
	CodeTimeout       = "TIMEOUT"
	CodeUnavailable   = "SERVICE_UNAVAILABLE"
	CodeInvalidInput  = "INVALID_INPUT"
)

// AppError represents an application error with HTTP status mapping
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	HTTPStatus int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"` // Original error for logging
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the original error for errors.Is/As support
func (e *AppError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code for this error
func (e *AppError) StatusCode() int {
	return e.HTTPStatus
}

// ToJSON converts the error to JSON bytes for HTTP responses
func (e *AppError) ToJSON() []byte {
	response := ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	}
	data, _ := json.Marshal(response)
	return data
}

// ErrorResponse is the JSON structure returned to clients
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// New creates a new AppError with custom code, message, and status
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an existing error with AppError context
func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// WithDetails adds additional details to an error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// Helper constructors for common error types

// NotFound creates a 404 Not Found error
func NotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NotFoundWithID creates a 404 error with resource ID
func NotFoundWithID(resource, id string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Details: map[string]interface{}{
			"resource": resource,
			"id":       id,
		},
	}
}

// Validation creates a 400 Validation Error
func Validation(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// InvalidInput creates a 400 Invalid Input error
func InvalidInput(message string) *AppError {
	return &AppError{
		Code:       CodeInvalidInput,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// Conflict creates a 409 Conflict error
func Conflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// Internal creates a 500 Internal Server Error
func Internal(message string, err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// Timeout creates a 504 Gateway Timeout error
func Timeout(message string) *AppError {
	return &AppError{
		Code:       CodeTimeout,
		Message:    message,
		HTTPStatus: http.StatusGatewayTimeout,
	}
}

// Unavailable creates a 503 Service Unavailable error
func Unavailable(service string) *AppError {
	return &AppError{
		Code:       CodeUnavailable,
		Message:    fmt.Sprintf("%s is temporarily unavailable", service),
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError tries to convert an error to AppError
// If the error is not an AppError, wraps it as Internal error
func AsAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Internal("An unexpected error occurred", err)
}
