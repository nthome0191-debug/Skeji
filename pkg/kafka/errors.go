package kafka

import (
	"errors"
	"fmt"
)

// Error types for Kafka operations
var (
	// ErrProducerClosed indicates the producer has been closed
	ErrProducerClosed = errors.New("kafka producer is closed")

	// ErrConsumerClosed indicates the consumer has been closed
	ErrConsumerClosed = errors.New("kafka consumer is closed")

	// ErrInvalidMessage indicates the message is invalid
	ErrInvalidMessage = errors.New("invalid message")

	// ErrEmptyKey indicates the message key is empty
	ErrEmptyKey = errors.New("message key cannot be empty")

	// ErrEmptyValue indicates the message value is empty
	ErrEmptyValue = errors.New("message value cannot be empty")

	// ErrMaxRetriesExceeded indicates max retries have been exceeded
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")

	// ErrTransientFailure indicates a transient failure that can be retried
	ErrTransientFailure = errors.New("transient failure")

	// ErrPermanentFailure indicates a permanent failure that should not be retried
	ErrPermanentFailure = errors.New("permanent failure")
)

// ErrorType represents the type of error
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error type
	ErrorTypeUnknown ErrorType = iota

	// ErrorTypeTransient represents a transient error (network issues, timeouts)
	ErrorTypeTransient

	// ErrorTypePermanent represents a permanent error (schema mismatch, invalid data)
	ErrorTypePermanent

	// ErrorTypeBusiness represents a business logic error
	ErrorTypeBusiness
)

// KafkaError wraps errors with additional context
type KafkaError struct {
	Type    ErrorType
	Message string
	Err     error
	Details map[string]interface{}
}

// Error implements the error interface
func (e *KafkaError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *KafkaError) Unwrap() error {
	return e.Err
}

// IsTransient checks if the error is transient (can be retried)
func (e *KafkaError) IsTransient() bool {
	return e.Type == ErrorTypeTransient
}

// IsPermanent checks if the error is permanent (should not be retried)
func (e *KafkaError) IsPermanent() bool {
	return e.Type == ErrorTypePermanent
}

// NewTransientError creates a new transient error
func NewTransientError(message string, err error) *KafkaError {
	return &KafkaError{
		Type:    ErrorTypeTransient,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// NewPermanentError creates a new permanent error
func NewPermanentError(message string, err error) *KafkaError {
	return &KafkaError{
		Type:    ErrorTypePermanent,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// NewBusinessError creates a new business logic error
func NewBusinessError(message string, err error) *KafkaError {
	return &KafkaError{
		Type:    ErrorTypeBusiness,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *KafkaError) WithDetail(key string, value interface{}) *KafkaError {
	e.Details[key] = value
	return e
}

// ClassifyError classifies an error as transient or permanent
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// Unwrap KafkaError
	var kafkaErr *KafkaError
	if errors.As(err, &kafkaErr) {
		return kafkaErr.Type
	}

	// Check for known transient errors
	errorMsg := err.Error()
	transientPatterns := []string{
		"connection refused",
		"timeout",
		"deadline exceeded",
		"no such host",
		"network is unreachable",
		"broken pipe",
		"connection reset",
		"i/o timeout",
		"temporary failure",
	}

	for _, pattern := range transientPatterns {
		if contains(errorMsg, pattern) {
			return ErrorTypeTransient
		}
	}

	// Check for known permanent errors
	permanentPatterns := []string{
		"invalid message",
		"schema mismatch",
		"deserialization failed",
		"unknown topic",
		"invalid configuration",
	}

	for _, pattern := range permanentPatterns {
		if contains(errorMsg, pattern) {
			return ErrorTypePermanent
		}
	}

	// Default to permanent if we can't classify it
	return ErrorTypePermanent
}

// ShouldRetry determines if an error should be retried
func ShouldRetry(err error, currentRetries, maxRetries int) bool {
	if err == nil {
		return false
	}

	if currentRetries >= maxRetries {
		return false
	}

	errorType := ClassifyError(err)
	return errorType == ErrorTypeTransient
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexSubstring(s, substr) >= 0)
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
