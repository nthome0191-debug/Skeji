package core

const (
	MAX_CONCURRENT_API_CALLS = 40
)

var RequestLimiter = make(chan struct{}, MAX_CONCURRENT_API_CALLS)

func ReqAcquire() { RequestLimiter <- struct{}{} }
func ReqRelease() { <-RequestLimiter }
