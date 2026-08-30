package greenhouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

func TestParseJobsResponse(t *testing.T) {
	body := loadFixture(t, "page1.json")
	jobs, err := parseJobsResponse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestMapJobNormalizesInternRole(t *testing.T) {
	body := loadFixture(t, "page1.json")
	jobs, err := parseJobsResponse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	cfg := Config{
		BoardToken:   "stripe",
		EmployerName: "Stripe",
		SourceURL:    "https://boards.greenhouse.io/stripe",
		Tags:         []string{"technology"},
	}

	raw, ok := mapJob(cfg, jobs[0])
	if !ok {
		t.Fatal("expected intern job to map")
	}
	if raw.ExternalID != "8031833" {
		t.Errorf("external_id = %q", raw.ExternalID)
	}
	if raw.Organization != "Stripe" {
		t.Errorf("organization = %q", raw.Organization)
	}
	if raw.ApplicationURL == "" || raw.SourceURL == "" {
		t.Error("expected application and source URLs")
	}
	if raw.Category != "internship" {
		t.Errorf("category = %q", raw.Category)
	}
	if raw.Description != "Build payments infrastructure as a summer intern." {
		t.Errorf("description = %q", raw.Description)
	}
}

func TestMapJobStripsEntityEncodedHTML(t *testing.T) {
	job := job{
		ID:          5108009008,
		Title:       "Applied Research Intern",
		AbsoluteURL: "http://block.xyz/careers/jobs/5108009008",
		Content:     "&lt;p&gt;&lt;strong&gt;Team:&lt;/strong&gt; Apollo — Block Applied R&amp;amp;D&lt;br&gt;&lt;strong&gt;Location:&lt;/strong&gt; Remote&lt;/p&gt;",
		Location:    &location{Name: "Remote"},
	}

	raw, ok := mapJob(Config{EmployerName: "Block"}, job)
	if !ok {
		t.Fatal("expected job to map")
	}
	want := "Team: Apollo — Block Applied R&D Location: Remote"
	if raw.Description != want {
		t.Errorf("description = %q, want %q", raw.Description, want)
	}
}

func TestFetchAllReturnsRelevantJobsFromSingleResponse(t *testing.T) {
	page1 := loadFixture(t, "page1.json")

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write(page1)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL + "/v1/boards",
		BoardToken:   "stripe",
		EmployerName: "Stripe",
		SourceURL:    "https://boards.greenhouse.io/stripe",
		PerPage:      100,
	})

	adapter := NewAdapter(server.Client())
	result, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected single HTTP request, got %d", requestCount)
	}
	if result.RawFetched != 2 {
		t.Fatalf("expected raw_fetched 2, got %d", result.RawFetched)
	}
	if result.FilteredOut != 1 {
		t.Fatalf("expected filtered_out 1, got %d", result.FilteredOut)
	}
	if len(result.Retained) != 1 {
		t.Fatalf("expected 1 relevant item from mixed response, got %d", len(result.Retained))
	}
}

func TestFetchAllIdempotentOnSecondRun(t *testing.T) {
	page1 := loadFixture(t, "page1.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(page1)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL + "/v1/boards",
		BoardToken:   "stripe",
		EmployerName: "Stripe",
		PerPage:      100,
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
		t.Fatalf("expected same count on second run, got %d then %d", len(first.Retained), len(second.Retained))
	}
}

func TestFetchAllPropagatesHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL + "/v1/boards",
		BoardToken:   "stripe",
		EmployerName: "Stripe",
	})

	adapter := NewAdapter(server.Client())
	_, err := adapter.FetchAll(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for failed upstream")
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

var _ = strconv.Itoa
