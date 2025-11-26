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
	defer func() {
		<-RequestLimiter
	}()
	fn()
}

func IsMissing(str string) bool {
	return len(str) == 0
}

func MissingParamErr(paramName string) error {
	return fmt.Errorf("required param [%v] is missing", paramName)
}
