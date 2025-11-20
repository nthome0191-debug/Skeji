package core

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
