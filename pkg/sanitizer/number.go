package sanitizer

const (
	MinPriority = 0

	MaxPriority = 100000
)

func NormalizePriority(priority int64) int64 {
	if priority < MinPriority {
		return MinPriority
	}
	if priority > MaxPriority {
		return MaxPriority
	}
	return priority
}
