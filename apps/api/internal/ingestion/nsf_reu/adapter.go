package nsf_reu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/opportunitytype"
)

const (
	defaultBaseURL        = "https://api.nsf.gov/services/v1/awards.json"
	defaultResultsPerPage = 25
	maxResultsPerPage     = 25
	awardShowURLFmt       = "https://www.nsf.gov/awardsearch/showAward?AWD_ID=%s"
)

var (
	urlPattern      = regexp.MustCompile(`https?://[^\s\)\]>"']+`)
	durationPattern = regexp.MustCompile(`(?i)(\d+)\s+weeks?`)
)

// HTTPDoer is satisfied by *http.Client for test injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds NSF REU adapter configuration.
type Config struct {
	Keyword         string `json:"keyword"`
	FundProgramName string `json:"fund_program_name"`
	ResultsPerPage  int    `json:"results_per_page"`
	BaseURL         string `json:"base_url"`
}

// Adapter fetches active NSF REU Site awards from the documented NSF Award API.
type Adapter struct {
	client HTTPDoer
}

// NewAdapter creates an NSF REU adapter.
func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &Adapter{client: client}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "nsf_reu" }

// FetchAll retrieves active REU Site awards and normalizes them to research opportunities.
func (a *Adapter) FetchAll(ctx context.Context, rawConfig json.RawMessage) (ingestrecord.FetchResult, error) {
	cfg, err := parseConfig(rawConfig)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}

	offset := 0
	totalCount := -1
	rawFetched := 0
	retained := make([]ingestrecord.RawOpportunity, 0)
	filterReasons := map[string]int{}

	for {
		body, count, fetchErr := a.fetchPage(ctx, cfg, offset)
		if fetchErr != nil {
			return ingestrecord.FetchResult{}, fetchErr
		}
		if totalCount < 0 {
			totalCount = count
		}

		awards, pageRaw, parseErr := parseAwardsResponse(body)
		if parseErr != nil {
			return ingestrecord.FetchResult{}, fmt.Errorf("parse NSF awards offset %d: %w", offset, parseErr)
		}
		rawFetched += pageRaw

		for _, award := range awards {
			raw, ok, reason := normalizeAward(award)
			if !ok {
				if reason != "" {
					filterReasons[reason]++
				}
				continue
			}
			retained = append(retained, raw)
		}

		offset += cfg.ResultsPerPage
		if totalCount >= 0 && offset >= totalCount {
			break
		}
		if pageRaw == 0 {
			break
		}
	}

	return ingestrecord.FetchResult{
		RawFetched:    rawFetched,
		Retained:      retained,
		FilteredOut:   rawFetched - len(retained),
		FilterReasons: filterReasons,
	}.MarkExhaustiveSuccess(), nil
}

func parseConfig(raw json.RawMessage) (Config, error) {
	cfg := Config{
		Keyword:         `"REU Site"`,
		FundProgramName: "RSCH EXPER FOR UNDERGRAD SITES",
		ResultsPerPage:  defaultResultsPerPage,
		BaseURL:         defaultBaseURL,
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse NSF REU config: %w", err)
		}
	}
	if cfg.ResultsPerPage < 1 {
		cfg.ResultsPerPage = defaultResultsPerPage
	}
	if cfg.ResultsPerPage > maxResultsPerPage {
		cfg.ResultsPerPage = maxResultsPerPage
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	return cfg, nil
}

func (a *Adapter) fetchPage(ctx context.Context, cfg Config, offset int) ([]byte, int, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid NSF base URL: %w", err)
	}

	q := u.Query()
	if cfg.Keyword != "" {
		q.Set("keyword", cfg.Keyword)
	}
	if cfg.FundProgramName != "" {
		q.Set("fundProgramName", cfg.FundProgramName)
	}
	q.Set("ActiveAwards", "True")
	q.Set("rpp", strconv.Itoa(cfg.ResultsPerPage))
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("NSF awards request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read NSF awards response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("NSF awards HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	count, err := extractTotalCount(body)
	if err != nil {
		return nil, 0, err
	}
	return body, count, nil
}

type nsfAward struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	AbstractText    string `json:"abstractText"`
	AwardeeName     string `json:"awardeeName"`
	AwardeeCity     string `json:"awardeeCity"`
	AwardeeStateCode string `json:"awardeeStateCode"`
	StartDate       string `json:"startDate"`
	ExpDate         string `json:"expDate"`
	Program         string `json:"program"`
	OrgLongName     string `json:"orgLongName"`
	FundProgramName string `json:"fundProgramName"`
	ActiveAwd       string `json:"activeAwd"`
}

func parseAwardsResponse(body []byte) ([]nsfAward, int, error) {
	var payload struct {
		Response struct {
			Award    []nsfAward `json:"award"`
			Metadata struct {
				TotalCount int `json:"totalCount"`
			} `json:"metadata"`
			ServiceNotification []struct {
				NotificationType    string `json:"notificationType"`
				NotificationMessage string `json:"notificationMessage"`
			} `json:"serviceNotification"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, 0, fmt.Errorf("unmarshal awards JSON: %w", err)
	}
	for _, n := range payload.Response.ServiceNotification {
		if strings.EqualFold(n.NotificationType, "ERROR") {
			return nil, 0, fmt.Errorf("NSF API error: %s", n.NotificationMessage)
		}
	}
	awards := payload.Response.Award
	if awards == nil {
		awards = []nsfAward{}
	}
	return awards, len(awards), nil
}

func extractTotalCount(body []byte) (int, error) {
	var payload struct {
		Response struct {
			Metadata struct {
				TotalCount int `json:"totalCount"`
			} `json:"metadata"`
			ServiceNotification []struct {
				NotificationType    string `json:"notificationType"`
				NotificationMessage string `json:"notificationMessage"`
			} `json:"serviceNotification"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("unmarshal NSF metadata: %w", err)
	}
	for _, n := range payload.Response.ServiceNotification {
		if strings.EqualFold(n.NotificationType, "ERROR") {
			return 0, fmt.Errorf("NSF API error: %s", n.NotificationMessage)
		}
	}
	return payload.Response.Metadata.TotalCount, nil
}

func normalizeAward(award nsfAward) (ingestrecord.RawOpportunity, bool, string) {
	title := strings.TrimSpace(award.Title)
	if title == "" {
		return ingestrecord.RawOpportunity{}, false, "missing_title"
	}
	if !strings.Contains(strings.ToLower(title), "reu site") {
		return ingestrecord.RawOpportunity{}, false, "not_reu_site_title"
	}
	org := strings.TrimSpace(award.AwardeeName)
	if org == "" {
		return ingestrecord.RawOpportunity{}, false, "missing_organization"
	}
	desc := strings.TrimSpace(award.AbstractText)
	if desc == "" {
		return ingestrecord.RawOpportunity{}, false, "missing_description"
	}
	if strings.TrimSpace(award.ID) == "" {
		return ingestrecord.RawOpportunity{}, false, "missing_external_id"
	}

	expDate, err := parseNSFDate(award.ExpDate)
	if err != nil {
		return ingestrecord.RawOpportunity{}, false, "invalid_exp_date"
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if expDate.Before(today) {
		return ingestrecord.RawOpportunity{}, false, "expired_award"
	}

	sourceURL := fmt.Sprintf(awardShowURLFmt, award.ID)
	links := classifyURLsFromAbstract(desc)

	location := formatLocation(award.AwardeeCity, award.AwardeeStateCode)
	meta := buildTypeMetadata(award, expDate, links)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return ingestrecord.RawOpportunity{}, false, "invalid_metadata"
	}

	return ingestrecord.RawOpportunity{
		ExternalID:         award.ID,
		Title:              title,
		Organization:       org,
		Description:        desc,
		Category:           "research",
		Location:           location,
		WorkArrangement:    "on_site",
		ApplicationURL:     "", // never set from NSF award-only discovery
		SourceURL:          sourceURL,
		Tags:               []string{"nsf", "reu", "undergraduate_research", "candidate_reu_site"},
		Skills:             []string{},
		OpportunityType:    opportunitytype.Research,
		TypeMetadata:       metaJSON,
		VerificationMethod: opportunitytype.VerificationOfficialSource,
		EducationLevel:     "undergraduate",
	}, true, ""
}

type researchMeta struct {
	ResearchArea                     *string `json:"research_area,omitempty"`
	DurationWeeks                    *int    `json:"duration_weeks,omitempty"`
	ProgramStart                     *string `json:"program_start,omitempty"`
	ProgramEnd                       *string `json:"program_end,omitempty"`
	ProgramURL                       *string `json:"program_url,omitempty"`
	ApplicationStatus                string  `json:"application_status"`
	ApplicationStatusMethod          string  `json:"application_status_method"`
	AvailabilityVerificationMethod   string  `json:"availability_verification_method"`
}

func buildTypeMetadata(award nsfAward, expDate time.Time, links classifiedURLs) researchMeta {
	meta := researchMeta{
		ApplicationStatus:              opportunitytype.ApplicationStatusUnknown,
		ApplicationStatusMethod:          opportunitytype.AvailabilityMethodNSFAwardOnly,
		AvailabilityVerificationMethod: opportunitytype.AvailabilityMethodUnknown,
	}

	if area := extractResearchArea(award.Title); area != "" {
		meta.ResearchArea = &area
	} else if org := strings.TrimSpace(award.OrgLongName); org != "" {
		meta.ResearchArea = &org
	}
	if weeks := extractDurationWeeks(award.AbstractText); weeks > 0 {
		meta.DurationWeeks = &weeks
	}
	if start, err := parseNSFDate(award.StartDate); err == nil {
		s := start.Format("2006-01-02")
		meta.ProgramStart = &s
	}
	end := expDate.Format("2006-01-02")
	meta.ProgramEnd = &end

	if links.ProgramURL != "" {
		meta.ProgramURL = &links.ProgramURL
	}

	return meta
}

func extractResearchArea(title string) string {
	const prefix = "REU Site:"
	lower := strings.ToLower(title)
	idx := strings.Index(lower, strings.ToLower(prefix))
	if idx < 0 {
		return ""
	}
	area := strings.TrimSpace(title[idx+len(prefix):])
	if area == "" {
		return ""
	}
	return area
}

func extractDurationWeeks(text string) int {
	match := durationPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	weeks, err := strconv.Atoi(match[1])
	if err != nil || weeks <= 0 {
		return 0
	}
	return weeks
}

func formatLocation(city, state string) string {
	city = strings.TrimSpace(city)
	state = strings.TrimSpace(state)
	switch {
	case city != "" && state != "":
		return fmt.Sprintf("%s, %s", titleCase(city), strings.ToUpper(state))
	case state != "":
		return strings.ToUpper(state)
	case city != "":
		return titleCase(city)
	default:
		return ""
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func parseNSFDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.Parse("01/02/2006", raw)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
