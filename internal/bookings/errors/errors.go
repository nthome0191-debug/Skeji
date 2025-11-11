package errors

import "errors"

var (
	ErrNotFound = errors.New("booking not found")

	ErrInvalidID = errors.New("invalid booking ID format")

	ErrTimeConflict = errors.New("booking time conflicts with existing booking")

	ErrCapacityExceeded = errors.New("booking capacity exceeded")

	ErrInvalidTimeRange = errors.New("end time must be after start time")
)
