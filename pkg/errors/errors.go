package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	CodeNotFound     = "NOT_FOUND"
	CodeValidation   = "VALIDATION_ERROR"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
	CodeBadRequest   = "BAD_REQUEST"
	CodeTimeout      = "TIMEOUT"
	CodeUnavailable  = "SERVICE_UNAVAILABLE"
	CodeInvalidInput = "INVALID_INPUT"
)

type AppError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	HTTPStatus int            `json:"-"`
	Details    map[string]any `json:"details,omitempty"`
	Err        error          `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) StatusCode() int {
	return e.HTTPStatus
}

func (e *AppError) ToJSON() []byte {
	response := ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	}
	data, _ := json.Marshal(response)
	return data
}

type ErrorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

func (e *AppError) WithDetails(details map[string]any) *AppError {
	e.Details = details
	return e
}

func NotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

func NotFoundWithID(resource, id string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Details: map[string]any{
			"resource": resource,
			"id":       id,
		},
	}
}

func Validation(message string, details map[string]any) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    details,
	}
}

func InvalidInput(message string) *AppError {
	return &AppError{
		Code:       CodeInvalidInput,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func Conflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

func Internal(message string, err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func Timeout(message string) *AppError {
	return &AppError{
		Code:       CodeTimeout,
		Message:    message,
		HTTPStatus: http.StatusGatewayTimeout,
	}
}

func Unavailable(service string) *AppError {
	return &AppError{
		Code:       CodeUnavailable,
		Message:    fmt.Sprintf("%s is temporarily unavailable", service),
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

func AsAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Internal("An unexpected error occurred", err)
}
