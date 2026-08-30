package ingestion

import (
	"strings"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/textutil"
)

// NormalizeCategory maps source-specific categories to CareerOS categories.
func NormalizeCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "internship", "intern":
		return "internship"
	case "full_time", "full-time", "fulltime":
		return "full_time"
	case "part_time", "part-time", "parttime":
		return "part_time"
	case "fellowship":
		return "fellowship"
	case "scholarship":
		return "scholarship"
	case "research":
		return "research"
	case "hackathon":
		return "hackathon"
	case "apprenticeship":
		return "apprenticeship"
	default:
		return "other"
	}
}

// NormalizeWorkArrangement maps source work arrangement values.
func NormalizeWorkArrangement(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "remote", "telework", "work from home":
		return "remote"
	case "hybrid":
		return "hybrid"
	default:
		return "on_site"
	}
}

// NormalizeRaw trims and validates a raw opportunity before persistence.
func NormalizeRaw(raw ingestrecord.RawOpportunity) (ingestrecord.RawOpportunity, bool) {
	raw.ExternalID = strings.TrimSpace(raw.ExternalID)
	raw.Title = strings.TrimSpace(raw.Title)
	raw.Organization = strings.TrimSpace(raw.Organization)
	raw.Description = textutil.PlainTextFromHTML(strings.TrimSpace(raw.Description))
	raw.Location = strings.TrimSpace(raw.Location)
	raw.ApplicationURL = strings.TrimSpace(raw.ApplicationURL)
	raw.SourceURL = strings.TrimSpace(raw.SourceURL)
	raw.Compensation = strings.TrimSpace(raw.Compensation)

	if raw.ExternalID == "" || raw.Title == "" || raw.Organization == "" || raw.Description == "" {
		return raw, false
	}
	if raw.ApplicationURL == "" && raw.SourceURL == "" {
		return raw, false
	}
	if raw.ApplicationURL == "" {
		raw.ApplicationURL = raw.SourceURL
	}
	if raw.SourceURL == "" {
		raw.SourceURL = raw.ApplicationURL
	}
	if !isPublicHTTPURL(raw.ApplicationURL) || !isPublicHTTPURL(raw.SourceURL) {
		return raw, false
	}

	raw.Category = NormalizeCategory(raw.Category)
	raw.WorkArrangement = NormalizeWorkArrangement(raw.WorkArrangement)
	if raw.Tags == nil {
		raw.Tags = []string{}
	}
	if raw.Skills == nil {
		raw.Skills = []string{}
	}
	if raw.ClassificationReasons == nil {
		raw.ClassificationReasons = []string{}
	}

	return raw, true
}
