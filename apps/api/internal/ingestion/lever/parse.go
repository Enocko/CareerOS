package lever

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/textutil"
)

type posting struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	HostedURL   string `json:"hostedUrl"`
	ApplyURL    string `json:"applyUrl"`
	CreatedAt   int64  `json:"createdAt"`
	WorkplaceType string `json:"workplaceType"`
	DescriptionPlain     string `json:"descriptionPlain"`
	DescriptionBodyPlain string `json:"descriptionBodyPlain"`
	Description          string `json:"description"`
	Categories  struct {
		Team        string   `json:"team"`
		Department  string   `json:"department"`
		Location    string   `json:"location"`
		Commitment  string   `json:"commitment"`
		AllLocations []string `json:"allLocations"`
	} `json:"categories"`
}

func parsePostingsResponse(body []byte) ([]posting, error) {
	var postings []posting
	if err := json.Unmarshal(body, &postings); err != nil {
		return nil, err
	}
	if postings == nil {
		return []posting{}, nil
	}
	return postings, nil
}

func mapPosting(cfg Config, posting posting) (ingestrecord.RawOpportunity, bool) {
	title := strings.TrimSpace(posting.Text)
	externalID := strings.TrimSpace(posting.ID)
	if title == "" || externalID == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	applicationURL := strings.TrimSpace(posting.ApplyURL)
	if applicationURL == "" {
		applicationURL = strings.TrimSpace(posting.HostedURL)
	}
	if applicationURL == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	sourceURL := strings.TrimSpace(posting.HostedURL)
	if sourceURL == "" {
		sourceURL = applicationURL
	}

	description := strings.TrimSpace(posting.DescriptionPlain)
	if description == "" {
		description = strings.TrimSpace(posting.DescriptionBodyPlain)
	}
	if description == "" {
		description = textutil.PlainTextFromHTML(posting.Description)
	}
	if description == "" {
		description = title
	}

	return ingestrecord.RawOpportunity{
		ExternalID:      externalID,
		Title:           title,
		Organization:    cfg.EmployerName,
		Description:     description,
		Category:        classifyCategory(posting),
		Location:        formatLocation(posting),
		WorkArrangement: classifyWorkArrangement(posting),
		Deadline:        nil,
		ApplicationURL:  applicationURL,
		SourceURL:       sourceURL,
		Tags:            buildTags(cfg, posting),
		Skills:          []string{},
	}, true
}

func formatLocation(posting posting) string {
	if locs := posting.Categories.AllLocations; len(locs) > 0 {
		parts := make([]string, 0, len(locs))
		for _, loc := range locs {
			if trimmed := strings.TrimSpace(loc); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}
	return strings.TrimSpace(posting.Categories.Location)
}

func buildTags(cfg Config, posting posting) []string {
	tags := append([]string{}, cfg.Tags...)
	tags = append(tags, "lever", "private-sector")
	if team := strings.TrimSpace(posting.Categories.Team); team != "" {
		tags = append(tags, team)
	}
	if dept := strings.TrimSpace(posting.Categories.Department); dept != "" {
		tags = append(tags, dept)
	}
	return uniqueStrings(tags)
}

func classifyCategory(posting posting) string {
	commitment := strings.ToLower(strings.TrimSpace(posting.Categories.Commitment))
	title := strings.ToLower(posting.Text)
	switch {
	case strings.Contains(commitment, "intern"):
		return "internship"
	case strings.Contains(title, "intern"):
		return "internship"
	case strings.Contains(title, "co-op") || strings.Contains(title, "co op"):
		return "internship"
	case strings.Contains(title, "new grad"):
		return "full_time"
	default:
		return "internship"
	}
}

func classifyWorkArrangement(posting posting) string {
	combined := strings.ToLower(posting.WorkplaceType + " " + posting.Categories.Location)
	switch {
	case strings.Contains(combined, "remote"):
		return "remote"
	case strings.Contains(combined, "hybrid"):
		return "hybrid"
	default:
		return "on_site"
	}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// ParseCreatedAt converts Lever createdAt epoch milliseconds.
func ParseCreatedAt(ms int64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

// ParsePostingsResponseForTest exposes parsePostingsResponse for unit tests.
func ParsePostingsResponseForTest(body []byte) ([]posting, error) {
	return parsePostingsResponse(body)
}

// MapPostingForTest exposes mapPosting for unit tests.
func MapPostingForTest(cfg Config, posting posting) (ingestrecord.RawOpportunity, bool) {
	return mapPosting(cfg, posting)
}
