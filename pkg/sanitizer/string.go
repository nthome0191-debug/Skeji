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

func NormalizeNameForComparison(name string) string {
	name = strings.TrimSpace(name)

	if name == "" {
		return ""
	}

	var result strings.Builder
	var lastWasSpace bool

	for _, r := range name {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			result.WriteRune(unicode.ToLower(r))
			lastWasSpace = false
		}
	}

	return result.String()
}

func NormalizeCity(city string) string {
	city = strings.TrimSpace(city)
	if city == "" {
		return ""
	}

	var result strings.Builder
	for _, r := range city {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

func NormalizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}

	var result strings.Builder
	for _, r := range label {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

func NormalizeWorkingDays(workingDays []string) []string {
	normalized := []string{}
	for _, wd := range workingDays {
		normalized = append(normalized, strings.TrimSpace(strings.ToLower(wd)))
	}
	return normalized
}
