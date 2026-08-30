package textutil

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlTagPattern    = regexp.MustCompile(`<[^>]+>`)
	blockBreakPattern = regexp.MustCompile(`(?i)</(p|div|li|h[1-6])>|<br\s*/?>`)
)

// PlainTextFromHTML converts HTML (including entity-encoded markup) to readable plain text.
func PlainTextFromHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Sources such as Greenhouse may return escaped tags (&lt;p&gt;) rather than raw HTML.
	for i := 0; i < 3; i++ {
		unescaped := html.UnescapeString(s)
		if unescaped == s {
			break
		}
		s = unescaped
	}

	s = blockBreakPattern.ReplaceAllString(s, " ")
	s = htmlTagPattern.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
