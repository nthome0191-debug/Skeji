package sanitizer

import (
	"strings"

	"github.com/nyaruka/phonenumbers"
)

var (
	supportedRegions = []string{
		"IL",
		"US",
	}
)

func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if phone == "" {
		return ""
	}

	for _, region := range supportedRegions {
		parsedNumber, err := phonenumbers.Parse(phone, region)
		if err == nil {
			return phonenumbers.Format(parsedNumber, phonenumbers.E164)
		}
	}
	return ""
}
