package sanitizer

const (
	MinPriority = 0

	MaxPriority = 100000
)

func NormalizePriority(priority int) int {
	if priority < MinPriority {
		return MinPriority
	}
	if priority > MaxPriority {
		return MaxPriority
	}
	return priority
}
