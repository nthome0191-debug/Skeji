package sanitizer

func NormalizeStringSlice(items []string, normalizer func(string, bool) string, keepDigits bool) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(items))

	for _, item := range items {
		normalized := normalizer(item, keepDigits)

		if normalized == "" {
			continue
		}

		if seen[normalized] {
			continue
		}

		seen[normalized] = true
		result = append(result, normalized)
	}

	return result
}

func NormalizeCities(cities []string) []string {
	return NormalizeStringSlice(cities, Normalize, false)
}

func NormalizeLabels(labels []string) []string {
	return NormalizeStringSlice(labels, Normalize, false)
}

func NormalizeExceptions(exp []string) []string {
	return NormalizeStringSlice(exp, Normalize, true)
}

// todo: maintainers should be changed to struct
// func NormalizeMaintainers(maintainers []string) []string {
// 	return NormalizeStringSlice(maintainers, NormalizePhone)
// }
