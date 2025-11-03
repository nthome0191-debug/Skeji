package sanitizer

import (
	"strings"
)

func NormalizeURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.SplitN(url, "/", 2)
	domain := strings.ToLower(parts[0])
	var path string
	if len(parts) > 1 {
		path = "/" + parts[1]
	}
	result := "https://" + domain + path
	result = strings.TrimSuffix(result, "/")
	return result
}
