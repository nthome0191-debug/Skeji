package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeValidation, "validation failed", http.StatusUnprocessableEntity)

	if err.Code != CodeValidation {
		t.Errorf("expected code %s, got %s", CodeValidation, err.Code)
	}
	if err.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %s", err.Message)
	}
	if err.HTTPStatus != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, err.HTTPStatus)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("database connection failed")
	wrapped := Wrap(originalErr, CodeInternal, "internal error", http.StatusInternalServerError)

	if wrapped.Err != originalErr {
		t.Errorf("expected wrapped error to contain original error")
	}
	if wrapped.Code != CodeInternal {
		t.Errorf("expected code %s, got %s", CodeInternal, wrapped.Code)
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name: "without underlying error",
			appErr: &AppError{
				Code:    CodeNotFound,
				Message: "resource not found",
			},
			expected: "NOT_FOUND: resource not found",
		},
		{
			name: "with underlying error",
			appErr: &AppError{
				Code:    CodeInternal,
				Message: "internal error",
				Err:     errors.New("database connection failed"),
			},
			expected: "INTERNAL_ERROR: internal error (caused by: database connection failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := Wrap(originalErr, CodeInternal, "wrapped", http.StatusInternalServerError)

	unwrapped := errors.Unwrap(appErr)
	if unwrapped != originalErr {
		t.Errorf("Unwrap() should return original error")
	}
}

func TestAppError_StatusCode(t *testing.T) {
	err := New(CodeNotFound, "not found", http.StatusNotFound)
	if err.StatusCode() != http.StatusNotFound {
		t.Errorf("StatusCode() = %d, want %d", err.StatusCode(), http.StatusNotFound)
	}
}

func TestAppError_WithDetails(t *testing.T) {
	err := New(CodeValidation, "validation failed", http.StatusUnprocessableEntity)
	details := map[string]any{
		"field": "email",
		"error": "invalid format",
	}

	err = err.WithDetails(details)

	if err.Details["field"] != "email" {
		t.Errorf("expected field 'email', got %v", err.Details["field"])
	}
	if err.Details["error"] != "invalid format" {
		t.Errorf("expected error 'invalid format', got %v", err.Details["error"])
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("User")

	if err.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.HTTPStatus)
	}
	if err.Message != "User not found" {
		t.Errorf("expected message 'User not found', got %s", err.Message)
	}
}

func TestNotFoundWithID(t *testing.T) {
	err := NotFoundWithID("User", "12345")

	if err.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.Details["id"] != "12345" {
		t.Errorf("expected id '12345', got %v", err.Details["id"])
	}
	if err.Details["resource"] != "User" {
		t.Errorf("expected resource 'User', got %v", err.Details["resource"])
	}
}

func TestValidation(t *testing.T) {
	details := map[string]any{"field": "email"}
	err := Validation("validation failed", details)

	if err.Code != CodeValidation {
		t.Errorf("expected code %s, got %s", CodeValidation, err.Code)
	}
	if err.HTTPStatus != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, err.HTTPStatus)
	}
	if err.Details["field"] != "email" {
		t.Errorf("expected field 'email', got %v", err.Details["field"])
	}
}

func TestInvalidInput(t *testing.T) {
	err := InvalidInput("invalid request")

	if err.Code != CodeInvalidInput {
		t.Errorf("expected code %s, got %s", CodeInvalidInput, err.Code)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("authentication required")

	if err.Code != CodeUnauthorized {
		t.Errorf("expected code %s, got %s", CodeUnauthorized, err.Code)
	}
	if err.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, err.HTTPStatus)
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")

	if err.Code != CodeForbidden {
		t.Errorf("expected code %s, got %s", CodeForbidden, err.Code)
	}
	if err.HTTPStatus != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, err.HTTPStatus)
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("resource already exists")

	if err.Code != CodeConflict {
		t.Errorf("expected code %s, got %s", CodeConflict, err.Code)
	}
	if err.HTTPStatus != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, err.HTTPStatus)
	}
}

func TestInternal(t *testing.T) {
	originalErr := errors.New("database error")
	err := Internal("internal error occurred", originalErr)

	if err.Code != CodeInternal {
		t.Errorf("expected code %s, got %s", CodeInternal, err.Code)
	}
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, err.HTTPStatus)
	}
	if err.Err != originalErr {
		t.Errorf("expected wrapped error to be originalErr")
	}
}

func TestTimeout(t *testing.T) {
	err := Timeout("request timed out")

	if err.Code != CodeTimeout {
		t.Errorf("expected code %s, got %s", CodeTimeout, err.Code)
	}
	if err.HTTPStatus != http.StatusGatewayTimeout {
		t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, err.HTTPStatus)
	}
}

func TestUnavailable(t *testing.T) {
	err := Unavailable("Payment Service")

	if err.Code != CodeUnavailable {
		t.Errorf("expected code %s, got %s", CodeUnavailable, err.Code)
	}
	if err.HTTPStatus != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, err.HTTPStatus)
	}
	if err.Message != "Payment Service is temporarily unavailable" {
		t.Errorf("expected message to contain service name, got %s", err.Message)
	}
}

func TestIsAppError(t *testing.T) {
	appErr := NotFound("User")
	regularErr := errors.New("regular error")

	if !IsAppError(appErr) {
		t.Errorf("IsAppError() should return true for AppError")
	}
	if IsAppError(regularErr) {
		t.Errorf("IsAppError() should return false for regular error")
	}
}

func TestAsAppError(t *testing.T) {
	appErr := NotFound("User")
	regularErr := errors.New("regular error")

	result := AsAppError(appErr)
	if result != appErr {
		t.Errorf("AsAppError() should return same AppError")
	}

	result = AsAppError(regularErr)
	if result.Code != CodeInternal {
		t.Errorf("AsAppError() should wrap regular error as internal error")
	}
	if result.Err != regularErr {
		t.Errorf("AsAppError() should wrap the original error")
	}
}

func TestAppError_ToJSON(t *testing.T) {
	err := NotFoundWithID("User", "12345")
	json := err.ToJSON()

	if len(json) == 0 {
		t.Errorf("ToJSON() should return non-empty JSON")
	}

	// Basic check that it contains expected fields
	jsonStr := string(json)
	if !contains(jsonStr, "NOT_FOUND") {
		t.Errorf("ToJSON() should contain error code")
	}
	if !contains(jsonStr, "not found") {
		t.Errorf("ToJSON() should contain error message")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
