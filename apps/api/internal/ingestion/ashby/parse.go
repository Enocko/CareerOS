package ashby

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/textutil"
)

type jobsResponse struct {
	APIVersion string `json:"apiVersion"`
	Jobs       []job  `json:"jobs"`
}

type job struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Department  string `json:"department"`
	Team        string `json:"team"`
	EmploymentType string `json:"employmentType"`
	Location    string `json:"location"`
	SecondaryLocations []struct {
		Location string `json:"location"`
	} `json:"secondaryLocations"`
	PublishedAt string `json:"publishedAt"`
	IsListed    bool   `json:"isListed"`
	IsRemote    bool   `json:"isRemote"`
	WorkplaceType string `json:"workplaceType"`
	JobURL      string `json:"jobUrl"`
	ApplyURL    string `json:"applyUrl"`
	DescriptionHTML  string `json:"descriptionHtml"`
	DescriptionPlain string `json:"descriptionPlain"`
	Compensation struct {
		CompensationTierSummary             *string `json:"compensationTierSummary"`
		ScrapeableCompensationSalarySummary *string `json:"scrapeableCompensationSalarySummary"`
	} `json:"compensation"`
}

func parseJobsResponse(body []byte) ([]job, error) {
	var resp jobsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Jobs == nil {
		return []job{}, nil
	}
	return resp.Jobs, nil
}

func mapJob(cfg Config, job job) (ingestrecord.RawOpportunity, bool) {
	title := strings.TrimSpace(job.Title)
	externalID := strings.TrimSpace(job.ID)
	if title == "" || externalID == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	applicationURL := strings.TrimSpace(job.ApplyURL)
	if applicationURL == "" {
		applicationURL = strings.TrimSpace(job.JobURL)
	}
	if applicationURL == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	sourceURL := strings.TrimSpace(job.JobURL)
	if sourceURL == "" {
		sourceURL = applicationURL
	}

	description := strings.TrimSpace(job.DescriptionPlain)
	if description == "" {
		description = textutil.PlainTextFromHTML(job.DescriptionHTML)
	}
	if description == "" {
		description = title
	}

	location := formatLocation(job)
	tags := buildTags(cfg, job)

	return ingestrecord.RawOpportunity{
		ExternalID:      externalID,
		Title:           title,
		Organization:    cfg.EmployerName,
		Description:     description,
		Category:        classifyCategory(job),
		Location:        location,
		WorkArrangement: classifyWorkArrangement(job),
		Deadline:        nil,
		ApplicationURL:  applicationURL,
		SourceURL:       sourceURL,
		Compensation:    formatCompensation(job),
		Tags:            tags,
		Skills:          []string{},
	}, true
}

func formatLocation(job job) string {
	primary := strings.TrimSpace(job.Location)
	secondary := make([]string, 0, len(job.SecondaryLocations))
	for _, loc := range job.SecondaryLocations {
		if name := strings.TrimSpace(loc.Location); name != "" {
			secondary = append(secondary, name)
		}
	}
	if primary == "" {
		return strings.Join(secondary, "; ")
	}
	if len(secondary) == 0 {
		return primary
	}
	return primary + "; " + strings.Join(secondary, "; ")
}

func buildTags(cfg Config, job job) []string {
	tags := append([]string{}, cfg.Tags...)
	tags = append(tags, "ashby", "private-sector")
	if dept := strings.TrimSpace(job.Department); dept != "" {
		tags = append(tags, dept)
	}
	if team := strings.TrimSpace(job.Team); team != "" {
		tags = append(tags, team)
	}
	return uniqueStrings(tags)
}

func classifyCategory(job job) string {
	employment := strings.ToLower(strings.TrimSpace(job.EmploymentType))
	title := strings.ToLower(job.Title)
	switch {
	case strings.Contains(employment, "intern"):
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

func classifyWorkArrangement(job job) string {
	combined := strings.ToLower(strings.TrimSpace(job.WorkplaceType + " " + job.Location))
	if job.IsRemote || strings.Contains(combined, "remote") {
		return "remote"
	}
	if strings.Contains(combined, "hybrid") {
		return "hybrid"
	}
	return "on_site"
}

func formatCompensation(job job) string {
	if job.Compensation.ScrapeableCompensationSalarySummary != nil {
		if summary := strings.TrimSpace(*job.Compensation.ScrapeableCompensationSalarySummary); summary != "" {
			return summary
		}
	}
	if job.Compensation.CompensationTierSummary != nil {
		if summary := strings.TrimSpace(*job.Compensation.CompensationTierSummary); summary != "" {
			return summary
		}
	}
	return ""
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

// ParsePublishedAt parses Ashby publishedAt when present.
func ParsePublishedAt(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}

// FormatLocation is exported for tests.
func FormatLocation(job job) string {
	return formatLocation(job)
}

// MapJobForTest exposes mapJob for unit tests.
func MapJobForTest(cfg Config, job job) (ingestrecord.RawOpportunity, bool) {
	return mapJob(cfg, job)
}

// ParseJobsResponseForTest exposes parseJobsResponse for unit tests.
func ParseJobsResponseForTest(body []byte) ([]job, error) {
	return parseJobsResponse(body)
}
