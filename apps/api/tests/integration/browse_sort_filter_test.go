package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// browseAllPages walks every page of a Browse query and returns the listings in order.
func browseAllPages(t *testing.T, router http.Handler, token, query string, perPage int) []map[string]any {
	t.Helper()
	var all []map[string]any
	for page := 1; ; page++ {
		url := fmt.Sprintf("/api/v1/opportunities?%s&per_page=%d&page=%d", query, perPage, page)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d body=%s", url, rec.Code, rec.Body.String())
		}
		var resp platform.PaginatedResponse[map[string]any]
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		all = append(all, resp.Data...)
		if page >= resp.Pagination.TotalPages {
			return all
		}
	}
}

func TestBrowseArrangementFilterAppliesToResearch(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	// Research programs are on-site; they used to leak into "remote only" on All.
	insertResearchWithApplicationStatus(t, pool, "open")
	insertSortableOpportunityWithTier(t, pool, "ARR-TEST Remote Intern", time.Now().UTC(), "high_confidence_technical", nil)

	for _, arrangement := range []string{"remote", "hybrid", "on_site"} {
		items := browseAllPages(t, router, token, "type=all&work_arrangement="+arrangement, 100)
		for _, item := range items {
			if got, _ := item["work_arrangement"].(string); got != arrangement {
				t.Fatalf("work_arrangement=%s returned %q listing %q (%v)",
					arrangement, got, item["title"], item["opportunity_type"])
			}
		}
	}
}

func TestBrowsePaginationIsStableWhenSortKeysTie(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	// Identical timestamps and no deadlines: every sort key ties, so only a
	// unique tiebreaker keeps LIMIT/OFFSET pages from overlapping.
	marker := "TIE" + uuid.NewString()[:8]
	seen := time.Now().UTC().Truncate(time.Second)
	words := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel",
		"India", "Juliet", "Kilo", "Lima", "Mike", "November", "Oscar", "Papa",
		"Quebec", "Romeo", "Sierra", "Tango", "Uniform", "Victor", "Whiskey", "Xray", "Yankee"}
	for _, w := range words {
		insertSortableOpportunityWithTier(t, pool, marker+" "+w+" Intern", seen, "high_confidence_technical", nil)
	}

	for _, sort := range []string{"newest", "deadline", "arrangement"} {
		for _, scope := range []string{"all", "employment"} {
			items := browseAllPages(t, router, token, "type="+scope+"&q="+marker+"&sort="+sort, 4)
			ids := map[string]bool{}
			for _, item := range items {
				id, _ := item["id"].(string)
				if ids[id] {
					t.Fatalf("type=%s sort=%s: listing %q appeared on more than one page", scope, sort, item["title"])
				}
				ids[id] = true
			}
			if len(ids) != len(words) {
				t.Fatalf("type=%s sort=%s: expected %d unique listings across pages, got %d", scope, sort, len(words), len(ids))
			}
		}
	}
}

func TestBrowseAllDeadlineSortPutsDeadlinesFirst(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	// An open research program (no deadline) used to outrank every job deadline
	// on the All tab because research status was the primary sort key.
	insertResearchWithApplicationStatus(t, pool, "open")
	deadline := time.Now().UTC().Add(3 * 24 * time.Hour)
	insertSortableOpportunityWithTier(t, pool, "DL-TEST Deadline Intern", time.Now().UTC(), "high_confidence_technical", &deadline)

	items := browseAllPages(t, router, token, "type=all&sort=deadline", 100)
	sawUndated := false
	var lastDeadline string
	for _, item := range items {
		d, _ := item["deadline"].(string)
		if d == "" {
			sawUndated = true
			continue
		}
		if sawUndated {
			t.Fatalf("listing with deadline %s appeared after listings without a deadline", d)
		}
		if d < lastDeadline {
			t.Fatalf("deadlines out of order: %s after %s", d, lastDeadline)
		}
		lastDeadline = d
	}
	if lastDeadline == "" {
		t.Fatal("expected the deadline fixture in results")
	}
}
