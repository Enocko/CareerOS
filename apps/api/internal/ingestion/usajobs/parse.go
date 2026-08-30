package usajobs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/textutil"
)

type searchResponse struct {
	SearchResult struct {
		SearchResultItems []searchResultItem `json:"SearchResultItems"`
		UserArea          struct {
			NumberOfPages string `json:"NumberOfPages"`
		} `json:"UserArea"`
	} `json:"SearchResult"`
}

type searchResultItem struct {
	MatchedObjectDescriptor matchedObject `json:"MatchedObjectDescriptor"`
}

type matchedObject struct {
	PositionID               string   `json:"PositionID"`
	PositionTitle            string   `json:"PositionTitle"`
	OrganizationName         string   `json:"OrganizationName"`
	DepartmentName           string   `json:"DepartmentName"`
	PositionURI              string   `json:"PositionURI"`
	ApplyURI                 []string `json:"ApplyURI"`
	PositionLocationDisplay  string   `json:"PositionLocationDisplay"`
	PositionFormattedDescription []struct {
		Label string `json:"Label"`
		Content string `json:"Content"`
	} `json:"PositionFormattedDescription"`
	PositionRemuneration []struct {
		MinimumRange string `json:"MinimumRange"`
		MaximumRange string `json:"MaximumRange"`
		RateIntervalCode string `json:"RateIntervalCode"`
	} `json:"PositionRemuneration"`
	PositionScheduleTypeCode string `json:"PositionScheduleTypeCode"`
	PositionOfferingTypeCode string `json:"PositionOfferingTypeCode"`
	PositionEndDate          string `json:"PositionEndDate"`
	TeleworkEligible         bool   `json:"TeleworkEligible"`
	RemoteIndicator          bool   `json:"RemoteIndicator"`
}

func extractTotalPages(body []byte) (int, error) {
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse USAJobs pagination: %w", err)
	}
	pagesStr := strings.TrimSpace(resp.SearchResult.UserArea.NumberOfPages)
	if pagesStr == "" {
		return 1, nil
	}
	pages, err := strconv.Atoi(pagesStr)
	if err != nil {
		return 1, nil
	}
	return pages, nil
}

func parseSearchResponse(body []byte) ([]ingestrecord.RawOpportunity, int, error) {
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, err
	}

	rawCount := len(resp.SearchResult.SearchResultItems)
	items := make([]ingestrecord.RawOpportunity, 0, rawCount)
	for _, item := range resp.SearchResult.SearchResultItems {
		raw, ok := mapItem(item.MatchedObjectDescriptor)
		if ok {
			items = append(items, raw)
		}
	}
	return items, rawCount, nil
}

func mapItem(m matchedObject) (ingestrecord.RawOpportunity, bool) {
	org := strings.TrimSpace(m.OrganizationName)
	if org == "" {
		org = strings.TrimSpace(m.DepartmentName)
	}

	title := strings.TrimSpace(m.PositionTitle)
	externalID := strings.TrimSpace(m.PositionID)
	if externalID == "" || title == "" || org == "" {
		return ingestrecord.RawOpportunity{}, false
	}

	description := buildDescription(m)
	if description == "" {
		description = title
	}

	sourceURL := strings.TrimSpace(m.PositionURI)
	applicationURL := ""
	if len(m.ApplyURI) > 0 {
		applicationURL = strings.TrimSpace(m.ApplyURI[0])
	}
	if applicationURL == "" {
		applicationURL = sourceURL
	}

	deadline := parseDate(m.PositionEndDate)
	category := mapCategory(m.PositionOfferingTypeCode, m.PositionScheduleTypeCode)
	workArrangement := mapWorkArrangement(m.RemoteIndicator, m.TeleworkEligible)
	compensation := mapCompensation(m.PositionRemuneration)

	tags := []string{"federal", "usajobs"}
	if strings.Contains(strings.ToLower(title), "intern") {
		tags = append(tags, "internship")
	}

	return ingestrecord.RawOpportunity{
		ExternalID:      externalID,
		Title:           title,
		Organization:    org,
		Description:     description,
		Category:        category,
		Location:        strings.TrimSpace(m.PositionLocationDisplay),
		WorkArrangement: workArrangement,
		Deadline:        deadline,
		ApplicationURL:  applicationURL,
		SourceURL:       sourceURL,
		Compensation:    compensation,
		Tags:            tags,
		Skills:          []string{},
	}, true
}

func buildDescription(m matchedObject) string {
	var parts []string
	for _, block := range m.PositionFormattedDescription {
		label := strings.TrimSpace(block.Label)
		content := strings.TrimSpace(textutil.PlainTextFromHTML(block.Content))
		if content == "" {
			continue
		}
		if label != "" {
			parts = append(parts, label+":\n"+content)
		} else {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func parseDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}

func mapCategory(offering, schedule string) string {
	offering = strings.ToLower(strings.TrimSpace(offering))
	schedule = strings.ToLower(strings.TrimSpace(schedule))
	if strings.Contains(offering, "intern") || strings.Contains(schedule, "intermittent") {
		return "internship"
	}
	if strings.Contains(schedule, "part") {
		return "part_time"
	}
	return "full_time"
}

func mapWorkArrangement(remote, telework bool) string {
	if remote {
		return "remote"
	}
	if telework {
		return "hybrid"
	}
	return "on_site"
}

func mapCompensation(items []struct {
	MinimumRange string `json:"MinimumRange"`
	MaximumRange string `json:"MaximumRange"`
	RateIntervalCode string `json:"RateIntervalCode"`
}) string {
	if len(items) == 0 {
		return ""
	}
	item := items[0]
	min := strings.TrimSpace(item.MinimumRange)
	max := strings.TrimSpace(item.MaximumRange)
	interval := strings.TrimSpace(item.RateIntervalCode)
	if min == "" && max == "" {
		return ""
	}
	if min != "" && max != "" && min != max {
		return fmt.Sprintf("$%s - $%s %s", min, max, interval)
	}
	if min != "" {
		return fmt.Sprintf("$%s %s", min, interval)
	}
	return fmt.Sprintf("$%s %s", max, interval)
}
