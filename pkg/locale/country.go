package locale

import (
	"strings"
)

const (
	DefaultTimezone = "UTC"
)

type Country struct {
	Code            string   // ISO 3166-1 alpha-2 country code (e.g., "IL", "US")
	Name            string   // Human-readable country name
	PhonePrefixes   []string // Valid phone number prefixes (e.g., ["+972", "972"])
	DefaultTimezone string   // IANA timezone identifier (e.g., "Asia/Jerusalem")
}

var (
	Countries = map[string]Country{
		"IL": {
			Code:            "IL",
			Name:            "Israel",
			PhonePrefixes:   []string{"+972", "972"},
			DefaultTimezone: "Asia/Jerusalem",
		},
		"US": {
			Code:            "US",
			Name:            "United States",
			PhonePrefixes:   []string{"+1", "1"},
			DefaultTimezone: "America/New_York",
		},
	}

	TimeZoneTags = map[string][]string{
		"IL": {"Asia/Jerusalem", "Israel", "Asia/Tel_Aviv"},
		"US": {"America/New_York", "America/Los_Angeles", "US/Eastern", "US/Pacific"},
	}
	SupportedTimeZones = map[string]bool{
		"Asia/Jerusalem":      true,
		"Asia/Tel_Aviv":       true,
		"America/New_York":    true,
		"America/Los_Angeles": true,
		"US/Eastern":          true,
		"US/Pacific":          true,
	}
)

func DetectRegion(tz string) string {
	for region, zones := range TimeZoneTags {
		for _, z := range zones {
			if strings.EqualFold(tz, z) {
				return region
			}
		}
	}
	return "IL"
}
