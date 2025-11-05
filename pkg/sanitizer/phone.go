package sanitizer

import (
	"strings"

	"github.com/nyaruka/phonenumbers"
)

func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return ""
	}

	supportedRegions := []string{"IL", "US"}

	for _, region := range supportedRegions {
		parsedNumber, err := phonenumbers.Parse(phone, region)
		if err != nil {
			continue
		}

		if phonenumbers.IsValidNumber(parsedNumber) {
			return phonenumbers.Format(parsedNumber, phonenumbers.E164)
		}
	}

	return ""
}
