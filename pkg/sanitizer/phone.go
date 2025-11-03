package sanitizer

import (
	"strings"
	"unicode"
)

func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return ""
	}

	var digits strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}

	normalized := digits.String()

	if normalized != "" {
		return "+" + normalized
	}

	return ""
}
