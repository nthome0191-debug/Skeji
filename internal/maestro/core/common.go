package core

import "fmt"

const (
	MAX_CONCURRENT_API_CALLS = 40
)

var (
	RequestLimiter = make(chan struct{}, MAX_CONCURRENT_API_CALLS)
)

func RunWithRateLimitedConcurrency(fn func()) {
	RequestLimiter <- struct{}{}

	var released bool
	defer func() {
		if !released {
			<-RequestLimiter
		}
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	defer func() {
		<-RequestLimiter
		released = true
	}()
	fn()
}

func IsMissing(str string) bool {
	return len(str) == 0
}

func MissingParamErr(paramName string) error {
	return fmt.Errorf("required param [%v] is missing", paramName)
}
