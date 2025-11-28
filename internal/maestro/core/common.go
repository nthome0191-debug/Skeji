package core

import "fmt"

const (
	MAX_CONCURRENT_API_CALLS = 40
)

var (
	RequestLimiter = make(chan struct{}, MAX_CONCURRENT_API_CALLS)
)

// RunWithRateLimitedConcurrency executes fn with rate limiting.
// Ensures the semaphore slot is always released, even if fn panics.
// This function guarantees slot release through careful defer ordering.
func RunWithRateLimitedConcurrency(fn func()) {
	// Acquire semaphore slot
	RequestLimiter <- struct{}{}

	// Double-layered protection against slot leaks:
	// 1. Outer defer with recover ensures slot release even on panic
	// 2. Inner defer releases slot in normal flow
	var released bool
	defer func() {
		if !released {
			// This is a safety net - should never be hit if everything works correctly
			// But if it is hit, we prevent permanent slot loss
			<-RequestLimiter
		}
		// Re-panic after releasing slot so caller's recover() can handle it
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	defer func() {
		// Normal slot release path
		<-RequestLimiter
		released = true
	}()

	// Execute the function
	fn()
}

func IsMissing(str string) bool {
	return len(str) == 0
}

func MissingParamErr(paramName string) error {
	return fmt.Errorf("required param [%v] is missing", paramName)
}
