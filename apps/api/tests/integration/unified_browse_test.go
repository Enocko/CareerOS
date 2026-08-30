package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUnifiedBrowseAllIncludesEmploymentAndResearch(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	_ = insertBrowsableEmploymentOpportunity(t, pool)
	_ = insertVerifiedResearchOpportunity(t, pool)

	for _, query := range []struct {
		q    string
		typ  string
	}{
		{q: "BrowseTestCorp", typ: "employment"},
		{q: "Quantum", typ: "research"},
	} {
		url := fmt.Sprintf("/api/v1/opportunities?type=all&q=%s&per_page=100", query.q)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", query.typ, rec.Code, rec.Body.String())
		}

		var resp platform.PaginatedResponse[map[string]any]
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		found := false
		for _, item := range resp.Data {
			if oppType, _ := item["opportunity_type"].(string); oppType == query.typ {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s opportunities in type=all browse for q=%s", query.typ, query.q)
		}
	}
}

func TestUnifiedBrowseEmploymentExcludesResearch(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	_ = insertVerifiedEmploymentOpportunity(t, pool)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=employment&per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range resp.Data {
		if item["id"] == researchID {
			t.Fatal("research opportunity leaked into employment browse")
		}
		if oppType, _ := item["opportunity_type"].(string); oppType != "employment" {
			t.Fatalf("expected employment only, got %q", oppType)
		}
	}
}

func TestUnifiedBrowseResearchExcludesEmployment(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	empID := insertVerifiedEmploymentOpportunity(t, pool)
	_ = insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=research&per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range resp.Data {
		if item["id"] == empID {
			t.Fatal("employment opportunity leaked into research browse")
		}
		if oppType, _ := item["opportunity_type"].(string); oppType != "research" {
			t.Fatalf("expected research only, got %q for %v", oppType, item["title"])
		}
	}
}

func TestUnifiedBrowseResearchUsesTypeAndOpportunityTypeParams(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	_ = insertVerifiedEmploymentOpportunity(t, pool)
	_ = insertVerifiedResearchOpportunity(t, pool)

	for _, query := range []string{
		"type=research",
		"type=research&opportunity_type=research",
		"opportunity_type=research",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?"+query+"&per_page=100", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("query %q: expected 200, got %d", query, rec.Code)
		}
		var resp platform.PaginatedResponse[map[string]any]
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("query %q decode: %v", query, err)
		}
		for _, item := range resp.Data {
			if oppType, _ := item["opportunity_type"].(string); oppType != "research" {
				t.Fatalf("query %q leaked %q opportunity: %v", query, oppType, item["title"])
			}
		}
	}
}

func TestUnifiedBrowseAllSearchReturnsBothTypes(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	insertVerifiedEmploymentOpportunity(t, pool)
	insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=all&q=Test&per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	types := map[string]bool{}
	for _, item := range resp.Data {
		if oppType, _ := item["opportunity_type"].(string); oppType != "" {
			types[oppType] = true
		}
	}
	if !types["employment"] || !types["research"] {
		t.Fatalf("expected both types in all-search results, got %v", types)
	}
}

func TestUnifiedBrowseAllExcludesUnverifiedDevSeed(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	_ = insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=all&per_page=500", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range resp.Data {
		if status, _ := item["verification_status"].(string); status != "verified" {
			t.Fatalf("unverified record leaked into all browse: %v", item["title"])
		}
		if source, _ := item["source_name"].(string); source == "dev_seed" {
			t.Fatalf("dev_seed record leaked into all browse: %v", item["title"])
		}
	}
}

func TestUnifiedBrowseSearchWithinResearchScope(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=research&opportunity_type=research&q=Quantum&per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, item := range resp.Data {
		if item["id"] == researchID {
			found = true
		}
		if oppType, _ := item["opportunity_type"].(string); oppType != "research" {
			t.Fatalf("research search leaked %q", oppType)
		}
	}
	if !found {
		t.Fatal("expected research search to return inserted REU by metadata/title match")
	}
}

func TestUnifiedBrowseSearchWithinEmploymentScope(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	empID := insertBrowsableEmploymentOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?type=employment&opportunity_type=employment&q=BrowseTestCorp&per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, item := range resp.Data {
		if item["id"] == empID {
			found = true
		}
		if oppType, _ := item["opportunity_type"].(string); oppType != "employment" {
			t.Fatalf("employment search leaked %q", oppType)
		}
	}
	if !found {
		t.Fatal("expected employment search to return inserted job by organization match")
	}
}

func insertBrowsableEmploymentOpportunity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("BROWSE-TEST-%s", uuid.NewString())
	now := time.Now().UTC()
	sourceID := "c3000000-0000-4000-8000-000000000001"
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, opportunity_type, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count,
			experience_level, career_family, education_level, relevance_tier, classification_reasons,
			type_metadata, verification_method
		) VALUES (
			$1, $2, $3, 'Browse Test Software Engineer Intern', 'BrowseTestCorp', 'Browsable test opportunity.',
			'internship', 'employment', 'remote', 'https://www.example.com/apply', 'https://www.example.com/job', 'TestSource',
			'verified', $4, $4, $4, 'open', '{python}', '{integration}', 0,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{internship,software_engineering}',
			'{}'::jsonb, 'official_source'
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert browsable employment opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func TestResearchUnknownNeverMarkedOpenInMetadata(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+researchID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	meta, _ := detail["type_metadata"].(map[string]any)
	if meta["application_status"] == "open" {
		t.Fatal("unknown research must not be labeled open")
	}
}
