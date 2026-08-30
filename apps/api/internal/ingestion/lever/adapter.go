package lever

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
	defaultUSBaseURL  = "https://api.lever.co/v0/postings"
	defaultEUBaseURL  = "https://api.eu.lever.co/v0/postings"
	defaultPageLimit  = 100
	defaultMaxRetries = 2
)

// HTTPDoer is satisfied by *http.Client for test injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds Lever board adapter configuration.
type Config struct {
	BoardToken   string   `json:"board_token"`
	EmployerName string   `json:"employer_name"`
	SourceURL    string   `json:"source_url"`
	Tags         []string `json:"tags"`
	BaseURL      string   `json:"base_url"`
	PageLimit    int      `json:"page_limit"`
}

// Adapter fetches opportunities from a Lever public job board.
type Adapter struct {
	client HTTPDoer
}

// NewAdapter creates a Lever adapter.
func NewAdapter(client HTTPDoer) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	return &Adapter{client: client}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "lever" }

// FetchAll retrieves listed public postings relevant to students.
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
		return ingestrecord.FetchResult{}, fmt.Errorf("lever board_token is required")
	}
	if cfg.EmployerName == "" {
		return ingestrecord.FetchResult{}, fmt.Errorf("lever employer_name is required")
	}

	postings, err := a.fetchAllPages(ctx, cfg)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}

	retained := make([]ingestrecord.RawOpportunity, 0)
	filteredOut := 0
	filterReasons := make(map[string]int)
	for _, posting := range postings {
		_, retain, filterReason := v2.EvaluateIngest(posting.Text, "")
		if !retain {
			filteredOut++
			relevance.RecordFilterReason(filterReasons, filterReason)
			continue
		}
		raw, ok := mapPosting(cfg, posting)
		if !ok {
			filteredOut++
			relevance.RecordFilterReason(filterReasons, relevance.ReasonUnknown)
			continue
		}
		raw.ApplyClassification()
		retained = append(retained, raw)
	}

	return ingestrecord.FetchResult{
		RawFetched:    len(postings),
		Retained:      retained,
		FilteredOut:   filteredOut,
		FilterReasons: filterReasons,
	}.MarkExhaustiveSuccess(), nil
}

// ListAllTitles returns every posting title without relevance filtering (audit use).
func (a *Adapter) ListAllTitles(ctx context.Context, cfg Config) ([]string, error) {
	postings, err := a.fetchAllPages(ctx, cfg)
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(postings))
	for _, posting := range postings {
		if t := strings.TrimSpace(posting.Text); t != "" {
			titles = append(titles, t)
		}
	}
	return titles, nil
}

func (a *Adapter) fetchAllPages(ctx context.Context, cfg Config) ([]posting, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultUSBaseURL
	}

	all := make([]posting, 0)
	skip := 0
	for {
		page, err := a.fetchPage(ctx, baseURL, cfg, skip)
		if err != nil {
			if skip == 0 && cfg.BaseURL == "" && isNotFound(err) {
				return a.fetchAllPagesWithBase(ctx, cfg, defaultEUBaseURL)
			}
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(page) < cfg.PageLimit {
			break
		}
		skip += cfg.PageLimit
	}

	return all, nil
}

func (a *Adapter) fetchAllPagesWithBase(ctx context.Context, cfg Config, baseURL string) ([]posting, error) {
	cfg.BaseURL = baseURL
	return a.fetchAllPages(ctx, cfg)
}

func (a *Adapter) fetchPage(ctx context.Context, baseURL string, cfg Config, skip int) ([]posting, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(cfg.BoardToken)))
	if err != nil {
		return nil, fmt.Errorf("invalid Lever URL: %w", err)
	}
	q := u.Query()
	q.Set("mode", "json")
	q.Set("skip", strconv.Itoa(skip))
	q.Set("limit", strconv.Itoa(cfg.PageLimit))
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
			lastErr = fmt.Errorf("Lever request failed: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read Lever response: %w", readErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return parsePostingsResponse(body)
		}

		lastErr = fmt.Errorf("Lever API returned status %d: %s", resp.StatusCode, truncate(string(body), 300))
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status 404")
}

func parseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse Lever config: %w", err)
		}
	}
	if cfg.PageLimit <= 0 {
		cfg.PageLimit = defaultPageLimit
	}
	if cfg.SourceURL == "" && cfg.BoardToken != "" {
		cfg.SourceURL = fmt.Sprintf("https://jobs.lever.co/%s", cfg.BoardToken)
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
