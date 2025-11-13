package sanitizer

import (
	"net/url"
	"regexp"
	"skeji/pkg/config"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

type Strategy func(string) string

type Pipeline []Strategy

func (p Pipeline) Apply(s string) string {
	for _, fn := range p {
		s = fn(s)
	}
	return s
}

var (
	reKeepLettersDigits = regexp.MustCompile(`[^0-9\p{L}]+`)
	reKeepLettersOnly   = regexp.MustCompile(`[^\p{L}]+`)
	reTrimUnderscores   = regexp.MustCompile(`_+`)

	supportedRegions = []string{
		"IL",
		"US",
	}
	reValidTZ         = regexp.MustCompile(`^[A-Za-z0-9_\-/]+$`)
	reMultiSlash      = regexp.MustCompile(`/+`)
	reMultiUnderscore = regexp.MustCompile(`_+`)

	reValidPhone = regexp.MustCompile(`^(?:|\+[1-9]\d{7,14})$`)
)

func trimAndLower(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}

func lower(s string) string {
	return strings.ToLower(s)
}

func collapseUnderscores(s string) string {
	s = reTrimUnderscores.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

func SanitizeNameOrAddress(input string) string {
	p := Pipeline{
		trimAndLower,
		func(s string) string { return reKeepLettersDigits.ReplaceAllString(s, "_") },
		collapseUnderscores,
		lower,
	}
	return p.Apply(input)
}

func SanitizeCityOrLabel(input string) string {
	p := Pipeline{
		trimAndLower,
		func(s string) string { return reKeepLettersOnly.ReplaceAllString(s, "_") },
		collapseUnderscores,
		lower,
	}
	return p.Apply(input)
}

func SanitizeSlice(values []string, strategy Strategy) []string {
	seen := make(map[string]struct{})
	out := []string{}

	for _, v := range values {
		s := strategy(v)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	return out
}

func SanitizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if phone == "" || !reValidPhone.MatchString(phone) {
		return phone
	}

	for _, region := range supportedRegions {
		parsedNumber, err := phonenumbers.Parse(phone, region)
		if err == nil {
			return phonenumbers.Format(parsedNumber, phonenumbers.E164)
		}
	}
	return ""
}

func SanitizePriority(cfg *config.Config, priority int64) int64 {
	if priority < int64(cfg.MinBusinessPriority) {
		return int64(cfg.MinBusinessPriority)
	}
	if priority > int64(cfg.MaxBusinessPriority) {
		return int64(cfg.MaxBusinessPriority)
	}
	return priority
}

func SanitizeURL(input string) string {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return ""
	}

	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(u.Host, "www."); ok {
		u.Host = after
	}

	u.Host = strings.TrimSuffix(u.Host, "/")
	u.Path = strings.TrimSuffix(strings.TrimSpace(u.Path), "/")

	q := u.Query()
	qClean := url.Values{}
	for k, v := range q {
		key := strings.TrimSpace(strings.ToLower(k))
		if strings.HasPrefix(key, "utm_") {
			continue
		}
		for _, val := range v {
			value := strings.TrimSpace(strings.ToLower(val))
			if value != "" {
				qClean.Add(key, value)
			}
		}
	}
	u.RawQuery = qClean.Encode()

	return u.String()
}

func SanitizeParticipantsMap(mp map[string]string) map[string]string {
	normalized := map[string]string{}
	for name, phone := range mp {
		name = SanitizeNameOrAddress(name)
		phone = SanitizePhone(phone)
		normalized[name] = phone
	}
	return normalized
}
