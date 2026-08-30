package lever

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

func TestParsePostingsResponse(t *testing.T) {
	body := loadFixture(t, "page1.json")
	postings, err := ParsePostingsResponseForTest(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(postings))
	}
}

func TestMapPostingNormalizesInternRole(t *testing.T) {
	body := loadFixture(t, "page1.json")
	postings, err := ParsePostingsResponseForTest(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	cfg := Config{
		BoardToken:   "demo",
		EmployerName: "Demo Co",
		SourceURL:    "https://jobs.lever.co/demo",
		Tags:         []string{"technology"},
	}

	raw, ok := MapPostingForTest(cfg, postings[0])
	if !ok {
		t.Fatal("expected intern posting to map")
	}
	if raw.ExternalID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("external_id = %q", raw.ExternalID)
	}
	if raw.Organization != "Demo Co" {
		t.Errorf("organization = %q", raw.Organization)
	}
	if raw.Category != "internship" {
		t.Errorf("category = %q", raw.Category)
	}
	if raw.WorkArrangement != "hybrid" {
		t.Errorf("work_arrangement = %q", raw.WorkArrangement)
	}
}

func TestFetchAllPaginatesMultiplePages(t *testing.T) {
	page1 := loadFixture(t, "page1.json")
	page2 := loadFixture(t, "page2.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skip := r.URL.Query().Get("skip")
		limit := r.URL.Query().Get("limit")
		if limit != "2" {
			http.Error(w, "unexpected limit", http.StatusBadRequest)
			return
		}
		switch skip {
		case "0":
			_, _ = w.Write(page1)
		case "2":
			_, _ = w.Write(page2)
		default:
			_, _ = w.Write([]byte("[]"))
		}
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "demo",
		EmployerName: "Demo Co",
		PageLimit:    2,
	})

	adapter := NewAdapter(server.Client())
	result, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if result.RawFetched != 3 {
		t.Fatalf("raw_fetched = %d, want 3", result.RawFetched)
	}
	if result.FilteredOut != 1 {
		t.Fatalf("filtered_out = %d, want 1", result.FilteredOut)
	}
	if len(result.Retained) != 2 {
		t.Fatalf("retained = %d, want 2", len(result.Retained))
	}
}

func TestFetchAllPropagatesHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "demo",
		EmployerName: "Demo Co",
	})

	adapter := NewAdapter(server.Client())
	_, err := adapter.FetchAll(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for failed upstream")
	}
}

func TestFetchAllMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[`))
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "demo",
		EmployerName: "Demo Co",
	})

	adapter := NewAdapter(server.Client())
	_, err := adapter.FetchAll(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
}

func TestFetchAllIdempotentOnSecondRun(t *testing.T) {
	page1 := loadFixture(t, "page1.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(page1)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:      server.URL,
		BoardToken:   "demo",
		EmployerName: "Demo Co",
		PageLimit:    100,
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

var _ = strconv.Itoa
