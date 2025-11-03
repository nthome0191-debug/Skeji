package sanitizer

func NormalizeStringSlice(items []string, normalizer func(string) string) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(items))

	for _, item := range items {
		normalized := normalizer(item)

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
	return NormalizeStringSlice(cities, NormalizeCity)
}

func NormalizeLabels(labels []string) []string {
	return NormalizeStringSlice(labels, NormalizeLabel)
}

func NormalizeMaintainers(maintainers []string) []string {
	return NormalizeStringSlice(maintainers, NormalizePhone)
}
