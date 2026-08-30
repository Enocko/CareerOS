package greenhouse

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
	"github.com/careeros/api/internal/ingestion/relevance"
	v2 "github.com/careeros/api/internal/ingestion/relevance/v2"
)

const (
	defaultBaseURL     = "https://boards-api.greenhouse.io/v1/boards"
	defaultPerPage     = 100
	maxPerPage         = 500
	defaultPageDelay   = 200 * time.Millisecond
	defaultMaxRetries  = 2
)

// HTTPDoer is satisfied by *http.Client for test injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds Greenhouse board adapter configuration.
type Config struct {
	BoardToken   string   `json:"board_token"`
	EmployerName string   `json:"employer_name"`
	SourceURL    string   `json:"source_url"`
	Tags         []string `json:"tags"`
	BaseURL      string   `json:"base_url"`
	PerPage      int      `json:"per_page"`
}

// Adapter fetches opportunities from a Greenhouse job board.
type Adapter struct {
	client HTTPDoer
}

// NewAdapter creates a Greenhouse adapter.
func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	return &Adapter{client: client}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "greenhouse" }

// FetchAll retrieves all relevant student opportunities from a Greenhouse board.
func (a *Adapter) FetchAll(ctx context.Context, rawConfig json.RawMessage) (ingestrecord.FetchResult, error) {
	cfg, err := parseConfig(rawConfig)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}
	return a.FetchAllWithConfig(ctx, cfg)
}

// FetchAllWithConfig fetches using a resolved config struct.
func (a *Adapter) FetchAllWithConfig(ctx context.Context, cfg Config) (ingestrecord.FetchResult, error) {
	if cfg.BoardToken == "" {
		return ingestrecord.FetchResult{}, fmt.Errorf("greenhouse board_token is required")
	}
	if cfg.EmployerName == "" {
		return ingestrecord.FetchResult{}, fmt.Errorf("greenhouse employer_name is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	// Greenhouse's public board API returns the full job list in a single response;
	// page/per_page are accepted but do not slice results for most boards.
	body, err := a.fetchPage(ctx, baseURL, cfg, 1)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}

	jobs, err := parseJobsResponse(body)
	if err != nil {
		return ingestrecord.FetchResult{}, fmt.Errorf("parse Greenhouse jobs: %w", err)
	}

	retained := make([]ingestrecord.RawOpportunity, 0)
	filteredOut := 0
	filterReasons := make(map[string]int)
	for _, job := range jobs {
		_, retain, filterReason := v2.EvaluateIngest(job.Title, "")
		if !retain {
			filteredOut++
			relevance.RecordFilterReason(filterReasons, filterReason)
			continue
		}
		raw, ok := mapJob(cfg, job)
		if !ok {
			filteredOut++
			relevance.RecordFilterReason(filterReasons, relevance.ReasonUnknown)
			continue
		}
		raw.ApplyClassification()
		retained = append(retained, raw)
	}

	return ingestrecord.FetchResult{
		RawFetched:    len(jobs),
		Retained:      retained,
		FilteredOut:   filteredOut,
		FilterReasons: filterReasons,
	}.MarkExhaustiveSuccess(), nil
}

// ListAllTitles returns every job title on the board without relevance filtering (audit use).
func (a *Adapter) ListAllTitles(ctx context.Context, cfg Config) ([]string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	body, err := a.fetchPage(ctx, baseURL, cfg, 1)
	if err != nil {
		return nil, err
	}
	jobs, err := parseJobsResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse Greenhouse jobs: %w", err)
	}
	titles := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if t := strings.TrimSpace(job.Title); t != "" {
			titles = append(titles, t)
		}
	}
	return titles, nil
}

func (a *Adapter) fetchPage(ctx context.Context, baseURL string, cfg Config, page int) ([]byte, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s/jobs", strings.TrimRight(baseURL, "/"), url.PathEscape(cfg.BoardToken)))
	if err != nil {
		return nil, fmt.Errorf("invalid Greenhouse URL: %w", err)
	}
	q := u.Query()
	q.Set("content", "true")
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(cfg.PerPage))
	u.RawQuery = q.Encode()

	var lastErr error
	for attempt := 0; attempt <= defaultMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "CareerOS-Ingestion/1.0")

		resp, err := a.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("Greenhouse request failed: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read Greenhouse response: %w", readErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		lastErr = fmt.Errorf("Greenhouse API returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func parseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse Greenhouse config: %w", err)
		}
	}
	if cfg.PerPage <= 0 {
		cfg.PerPage = defaultPerPage
	}
	if cfg.PerPage > maxPerPage {
		cfg.PerPage = maxPerPage
	}
	if cfg.SourceURL == "" && cfg.BoardToken != "" {
		cfg.SourceURL = fmt.Sprintf("https://boards.greenhouse.io/%s", cfg.BoardToken)
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{}
	}
	return cfg, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
