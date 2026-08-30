package ingestion

import (
	"net/url"
	"strings"
)

// isPublicHTTPURL rejects non-http(s) URLs and obvious local/test hosts.
func isPublicHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".local") {
		return false
	}
	return true
}
