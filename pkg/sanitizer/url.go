package sanitizer

import (
	"net/url"
	"strings"
)

func NormalizeURL(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	u, err := url.Parse(input)
	if err != nil {
		return "invalid_url"
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for _, key := range []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term",
		"utm_content", "fbclid", "gclid", "ref", "ref_src",
	} {
		q.Del(key)
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	if after, ok := strings.CutPrefix(u.Host, "www."); ok {
		u.Host = after
	}
	u.Path = strings.ToLower(u.Path)
	normalized := u.Scheme + "://" + u.Host + u.Path
	if u.RawQuery != "" {
		normalized += "?" + u.RawQuery
	}
	return normalized
}
