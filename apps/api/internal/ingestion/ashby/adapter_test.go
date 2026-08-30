package ashby

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseJobsResponse(t *testing.T) {
	body := loadFixture(t, "jobs.json")
	jobs, err := ParseJobsResponseForTest(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestMapJobNormalizesInternRole(t *testing.T) {
	body := loadFixture(t, "jobs.json")
	jobs, err := ParseJobsResponseForTest(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	cfg := Config{
		BoardToken:   "notion",
		EmployerName: "Notion",
		SourceURL:    "https://jobs.ashbyhq.com/notion",
		Tags:         []string{"technology"},
	}

	raw, ok := MapJobForTest(cfg, jobs[0])
	if !ok {
		t.Fatal("expected intern job to map")
	}
	if raw.ExternalID != "e66c6658-9e65-4c58-8db2-844628b6e8f8" {
		t.Errorf("external_id = %q", raw.ExternalID)
	}
	if raw.Organization != "Notion" {
		t.Errorf("organization = %q", raw.Organization)
	}
	if raw.Location != "San Francisco, California; New York, New York" {
		t.Errorf("location = %q", raw.Location)
	}
	if raw.WorkArrangement != "hybrid" {
		t.Errorf("work_arrangement = %q", raw.WorkArrangement)
	}
	if raw.Description != "Build collaborative software as a winter intern." {
		t.Errorf("description = %q", raw.Description)
	}
}

func TestFetchAllFiltersListedAndRelevance(t *testing.T) {
	fixture := loadFixture(t, "jobs.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("includeCompensation") != "true" {
			http.Error(w, "missing compensation flag", http.StatusBadRequest)
			return
		}
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "notion",
		EmployerName: "Notion",
		SourceURL:    "https://jobs.ashbyhq.com/notion",
	})

	adapter := NewAdapter(server.Client())
	result, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if result.RawFetched != 3 {
		t.Fatalf("raw_fetched = %d, want 3", result.RawFetched)
	}
	if result.FilteredOut != 2 {
		t.Fatalf("filtered_out = %d, want 2", result.FilteredOut)
	}
	if len(result.Retained) != 1 {
		t.Fatalf("retained = %d, want 1", len(result.Retained))
	}
}

func TestFetchAllPropagatesHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "notion",
		EmployerName: "Notion",
	})

	adapter := NewAdapter(server.Client())
	_, err := adapter.FetchAll(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for failed upstream")
	}
}

func TestFetchAllMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jobs":`))
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "notion",
		EmployerName: "Notion",
	})

	adapter := NewAdapter(server.Client())
	_, err := adapter.FetchAll(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
}

func TestFetchAllIdempotentOnSecondRun(t *testing.T) {
	fixture := loadFixture(t, "jobs.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "notion",
		EmployerName: "Notion",
	})

	adapter := NewAdapter(server.Client())
	first, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("first FetchAll: %v", err)
	}
	second, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("second FetchAll: %v", err)
	}
	if len(first.Retained) != len(second.Retained) {
		t.Fatalf("expected same retained count, got %d then %d", len(first.Retained), len(second.Retained))
	}
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}
