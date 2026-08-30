package greenhouse

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/textutil"
)

type jobsResponse struct {
	Jobs []job `json:"jobs"`
}

type job struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	UpdatedAt   string    `json:"updated_at"`
	AbsoluteURL string    `json:"absolute_url"`
	Content     string    `json:"content"`
	Location    *location `json:"location"`
	Departments []struct {
		Name string `json:"name"`
	} `json:"departments"`
	Metadata []struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"metadata"`
}

type location struct {
	Name string `json:"name"`
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
	if title == "" || job.ID == 0 {
		return ingestrecord.RawOpportunity{}, false
	}

	applicationURL := strings.TrimSpace(job.AbsoluteURL)
	if applicationURL == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	description := textutil.PlainTextFromHTML(job.Content)
	if description == "" {
		description = title
	}

	locationName := ""
	if job.Location != nil {
		locationName = strings.TrimSpace(job.Location.Name)
	}

	tags := append([]string{}, cfg.Tags...)
	tags = append(tags, "greenhouse", "private-sector")

	category := classifyCategory(title)
	workArrangement := classifyWorkArrangement(locationName, title, job.Metadata)

	return ingestrecord.RawOpportunity{
		ExternalID:      strconv.Itoa(job.ID),
		Title:           title,
		Organization:    cfg.EmployerName,
		Description:     description,
		Category:        category,
		Location:        locationName,
		WorkArrangement: workArrangement,
		Deadline:        nil,
		ApplicationURL:  applicationURL,
		SourceURL:       applicationURL,
		Tags:            uniqueStrings(tags),
		Skills:          []string{},
	}, true
}

func classifyCategory(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "intern"):
		return "internship"
	case strings.Contains(lower, "co-op") || strings.Contains(lower, "co op"):
		return "internship"
	case strings.Contains(lower, "new grad"):
		return "full_time"
	default:
		return "internship"
	}
}

func classifyWorkArrangement(location, title string, metadata []struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}) string {
	combined := strings.ToLower(location + " " + title)
	for _, item := range metadata {
		combined += " " + strings.ToLower(fmt.Sprint(item.Value))
	}
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

// ParseUpdatedAt parses Greenhouse updated_at when present.
func ParseUpdatedAt(raw string) *time.Time {
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
