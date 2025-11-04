package errors

import "errors"

var (
	ErrNotFound = errors.New("business unit not found")

	ErrInvalidID = errors.New("invalid business unit ID format")
)
