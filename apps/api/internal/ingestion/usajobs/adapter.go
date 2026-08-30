package usajobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
)

const (
	defaultBaseURL        = "https://data.usajobs.gov/api/Search"
	defaultResultsPerPage = 100
	maxResultsPerPage     = 500
)

// HTTPDoer is satisfied by *http.Client for test injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds USAJobs adapter configuration.
type Config struct {
	Keyword        string `json:"keyword"`
	HiringPath     string `json:"hiring_path"`
	ResultsPerPage int    `json:"results_per_page"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key,omitempty"`
	UserAgent      string `json:"user_agent,omitempty"`
}

// Adapter fetches opportunities from the USAJobs Search API.
type Adapter struct {
	client HTTPDoer
}

// NewAdapter creates a USAJobs adapter.
func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{client: client}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "usajobs" }

// FetchAll retrieves all pages from USAJobs for the configured search.
func (a *Adapter) FetchAll(ctx context.Context, rawConfig json.RawMessage) (ingestrecord.FetchResult, error) {
	cfg, err := parseConfig(rawConfig)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}
	if cfg.APIKey == "" {
		return ingestrecord.FetchResult{}, fmt.Errorf("USAJOBS_API_KEY is required")
	}
	if cfg.UserAgent == "" {
		return ingestrecord.FetchResult{}, fmt.Errorf("USAJOBS_USER_AGENT is required")
	}

	page := 1
	totalPages := 1
	all := make([]ingestrecord.RawOpportunity, 0)
	rawFetched := 0

	for page <= totalPages {
		body, pages, fetchErr := a.fetchPage(ctx, cfg, page)
		if fetchErr != nil {
			return ingestrecord.FetchResult{}, fetchErr
		}
		if page == 1 {
			totalPages = pages
			if totalPages < 1 {
				totalPages = 1
			}
		}

		items, pageRaw, parseErr := parseSearchResponse(body)
		if parseErr != nil {
			return ingestrecord.FetchResult{}, fmt.Errorf("parse USAJobs page %d: %w", page, parseErr)
		}
		rawFetched += pageRaw
		all = append(all, items...)
		page++
	}

	return ingestrecord.FetchResult{
		RawFetched:  rawFetched,
		Retained:    all,
		FilteredOut: rawFetched - len(all),
	}.MarkExhaustiveSuccess(), nil
}

func (a *Adapter) fetchPage(ctx context.Context, cfg Config, page int) ([]byte, int, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid USAJobs base URL: %w", err)
	}

	q := u.Query()
	if cfg.Keyword != "" {
		q.Set("Keyword", cfg.Keyword)
	}
	if cfg.HiringPath != "" {
		q.Set("HiringPath", cfg.HiringPath)
	}
	q.Set("Page", strconv.Itoa(page))
	q.Set("ResultsPerPage", strconv.Itoa(cfg.ResultsPerPage))
	q.Set("Fields", "Full")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create USAJobs request: %w", err)
	}
	req.Header.Set("Authorization-Key", cfg.APIKey)
	req.Host = "data.usajobs.gov"
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("USAJobs request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read USAJobs response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("USAJobs API returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	pages, err := extractTotalPages(body)
	if err != nil {
		return nil, 0, err
	}

	return body, pages, nil
}

func parseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse USAJobs config: %w", err)
		}
	}
	if cfg.ResultsPerPage <= 0 {
		cfg.ResultsPerPage = defaultResultsPerPage
	}
	if cfg.ResultsPerPage > maxResultsPerPage {
		cfg.ResultsPerPage = maxResultsPerPage
	}
	return cfg, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// MergeCredentials attaches runtime secrets to source config.
func MergeCredentials(raw json.RawMessage, apiKey, userAgent string) (Config, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return Config{}, err
	}
	cfg.APIKey = strings.TrimSpace(apiKey)
	cfg.UserAgent = strings.TrimSpace(userAgent)
	return cfg, nil
}

// FetchAllWithConfig fetches using a fully resolved config struct.
func (a *Adapter) FetchAllWithConfig(ctx context.Context, cfg Config) (ingestrecord.FetchResult, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}
	return a.FetchAll(ctx, encoded)
}
