package errors

import "errors"

var (
	ErrNotFound = errors.New("schedule not found")

	ErrInvalidID = errors.New("invalid schedule ID format")
)
