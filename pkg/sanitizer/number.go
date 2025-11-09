package sanitizer

import "skeji/pkg/config"

func NormalizePriority(cfg *config.Config, priority int64) int64 {
	if priority < int64(cfg.MinBusinessPriotity) {
		return int64(cfg.MinBusinessPriotity)
	}
	if priority > int64(cfg.MaxBusinessPriority) {
		return int64(cfg.MaxBusinessPriority)
	}
	return priority
}
