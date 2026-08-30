package usajobs

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

func TestParseSearchResponse(t *testing.T) {
	body := loadFixture(t, "page1.json")
	items, rawCount, err := parseSearchResponse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rawCount != 1 {
		t.Fatalf("expected raw count 1, got %d", rawCount)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ExternalID != "TEST-POS-001" {
		t.Errorf("external id = %q", items[0].ExternalID)
	}
	if items[0].ApplicationURL == "" {
		t.Error("expected application URL")
	}
	if items[0].SourceURL == "" {
		t.Error("expected source URL")
	}
}

func TestFetchAllPaginatesAllPages(t *testing.T) {
	page1 := loadFixture(t, "page1.json")
	page2 := loadFixture(t, "page2.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("Page")
		switch page {
		case "1":
			_, _ = w.Write(page1)
		case "2":
			_, _ = w.Write(page2)
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:        server.URL,
		Keyword:        "intern",
		HiringPath:     "student",
		ResultsPerPage: 1,
		APIKey:         "test-key",
		UserAgent:      "test@example.com",
	})

	adapter := NewAdapter(server.Client())
	result, err := adapter.FetchAll(t.Context(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(result.Retained) != 2 {
		t.Fatalf("expected 2 items across pages, got %d", len(result.Retained))
	}
}

func TestFetchAllPropagatesHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg, _ := json.Marshal(Config{
		BaseURL:   server.URL,
		APIKey:    "test-key",
		UserAgent: "test@example.com",
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

func TestExtractTotalPages(t *testing.T) {
	body := loadFixture(t, "page1.json")
	pages, err := extractTotalPages(body)
	if err != nil {
		t.Fatalf("extractTotalPages: %v", err)
	}
	if pages != 2 {
		t.Errorf("expected 2 pages, got %d", pages)
	}
}

func TestExtractTotalPagesDefaultsToOne(t *testing.T) {
	body := []byte(`{"SearchResult":{"UserArea":{}}}`)
	pages, err := extractTotalPages(body)
	if err != nil {
		t.Fatalf("extractTotalPages: %v", err)
	}
	if pages != 1 {
		t.Errorf("expected 1 page default, got %d", pages)
	}
}

// Ensure strconv used in adapter tests compiles when optimized away.
var _ = strconv.Itoa
