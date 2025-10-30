package locale

import "strings"

func InferTimezoneFromPhone(phone string) string {
	normalized := strings.TrimSpace(phone)

	for _, country := range Countries {
		for _, prefix := range country.PhonePrefixes {
			if strings.HasPrefix(normalized, prefix) {
				return country.DefaultTimezone
			}
		}
	}

	return DefaultTimezone
}

func InferCountryFromPhone(phone string) *Country {
	normalized := strings.TrimSpace(phone)

	for _, country := range Countries {
		for _, prefix := range country.PhonePrefixes {
			if strings.HasPrefix(normalized, prefix) {
				return &country
			}
		}
	}

	return nil
}
