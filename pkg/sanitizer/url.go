package sanitizer

import (
	"strings"
)

func NormalizeURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	url = strings.ToLower(url)
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = "https://" + url
	url = strings.TrimSuffix(url, "/")
	return url
}
