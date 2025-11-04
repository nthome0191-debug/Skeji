package repository

import "errors"

var (
	// ErrNotFound is returned when a business unit is not found by ID
	ErrNotFound = errors.New("business unit not found")

	// ErrInvalidID is returned when an ID format is invalid
	ErrInvalidID = errors.New("invalid business unit ID format")
)
