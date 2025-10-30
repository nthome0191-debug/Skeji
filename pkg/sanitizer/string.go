package sanitizer

import (
	"strings"
	"unicode"
)

func TrimAndNormalize(s string) string {
	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	var result strings.Builder
	var lastWasSpace bool

	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			result.WriteRune(r)
			lastWasSpace = false
		}
	}

	return result.String()
}

func NormalizeName(name string) string {
	return TrimAndNormalize(name)
}

func NormalizeCity(city string) string {
	return TrimAndNormalize(city)
}

func NormalizeLabel(label string) string {
	normalized := TrimAndNormalize(label)
	return strings.ToLower(normalized)
}
