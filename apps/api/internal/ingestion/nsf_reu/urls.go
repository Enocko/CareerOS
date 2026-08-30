package nsf_reu

import "strings"

// classifiedURLs separates authoritative links from unverified application claims.
type classifiedURLs struct {
	ProgramURL     string // official program/info page when found in abstract
	ApplicationURL string // only set when a specific application page is verified (never from award-only ingest)
}

func classifyURLsFromAbstract(abstract string) classifiedURLs {
	urls := urlPattern.FindAllString(abstract, -1)
	result := classifiedURLs{}

	for _, raw := range urls {
		u := strings.TrimRight(raw, ".,;)")
		if isNSFAwardURL(u) || isGenericETAPURL(u) {
			continue
		}
		if result.ProgramURL == "" {
			result.ProgramURL = u
		}
	}
	return result
}

func isNSFAwardURL(u string) bool {
	lower := strings.ToLower(u)
	return strings.Contains(lower, "nsf.gov/awardsearch") ||
		strings.Contains(lower, "nsf.gov/award") ||
		strings.Contains(lower, "www.nsf.gov/award")
}

func isGenericETAPURL(u string) bool {
	lower := strings.ToLower(strings.TrimRight(u, "/"))
	return lower == "https://etap.nsf.gov" || lower == "http://etap.nsf.gov"
}

// isETAPHost reports whether the URL is on ETAP (generic or specific path).
func isETAPHost(u string) bool {
	return strings.Contains(strings.ToLower(u), "etap.nsf.gov")
}
