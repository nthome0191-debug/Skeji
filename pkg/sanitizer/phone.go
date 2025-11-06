package sanitizer

import (
	"fmt"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

func NormalizePhone(phone string) string {
	fmt.Printf("natali print phone %s\n", phone)
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
		fmt.Printf("natali: parsed number: %s\n", parsedNumber)
		if phonenumbers.IsValidNumber(parsedNumber) {
			return phonenumbers.Format(parsedNumber, phonenumbers.E164)
		} else {
			fmt.Printf("not valid number!\n")
		}
	}

	return ""
}
