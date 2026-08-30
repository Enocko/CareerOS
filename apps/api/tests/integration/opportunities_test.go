package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOpportunitiesListAndDetail(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var listResp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}

	if listResp.Pagination.Total < 1 {
		t.Fatal("expected at least 1 verified opportunity")
	}
	if len(listResp.Data) < 1 {
		t.Fatal("expected at least 1 opportunity in data")
	}

	first := listResp.Data[0]
	if first["verification_status"] != ingestion.VerificationVerified {
		t.Errorf("expected verified status in list, got %v", first["verification_status"])
	}

	firstID, ok := first["id"].(string)
	if !ok || firstID == "" {
		t.Fatal("expected opportunity id in list response")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?category=internship", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("filter: expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+firstID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detail: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail["description"] == nil || detail["description"] == "" {
		t.Error("expected description in detail response")
	}
	if detail["verification_status"] != ingestion.VerificationVerified {
		t.Errorf("expected verified detail, got %v", detail["verification_status"])
	}
	source, ok := detail["source"].(map[string]any)
	if !ok || source["name"] == "" {
		t.Error("expected source attribution in detail response")
	}
	if _, ok := detail["last_checked_at"]; !ok {
		t.Error("expected last_checked_at in detail response")
	}
	if _, ok := detail["is_saved"].(bool); !ok {
		t.Error("expected is_saved boolean in detail response")
	}
	if _, ok := detail["has_application"].(bool); !ok {
		t.Error("expected has_application boolean in detail response")
	}
}

func TestOpportunitiesHidesUnverifiedSeedByDefault(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?include_unverified=true", nil)
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
	if resp.Pagination.Total < 10 {
		t.Errorf("expected seed opportunities with include_unverified, got total %d", resp.Pagination.Total)
	}
}

func insertVerifiedOpportunity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("API-TEST-%s", uuid.NewString())
	now := time.Now().UTC()
	sourceID := "c3000000-0000-4000-8000-000000000001"
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count,
			experience_level, career_family, education_level, relevance_tier, classification_reasons
		) VALUES (
			$1, $2, $3, 'Software Engineer Intern', 'Test Agency', 'Verified test opportunity.',
			'internship', 'remote', 'https://www.usajobs.gov/test', 'https://www.usajobs.gov/test', 'USAJobs',
			'verified', $4, $4, $4, 'open', '{}', '{integration}', 0,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{internship,software_engineering}'
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert verified opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func TestOpportunitiesNotFound(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/00000000-0000-4000-8000-000000000099", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestOpportunitiesUnauthorized(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestOpportunitiesPagination(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	insertVerifiedOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?per_page=1&page=1", nil)
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
	if len(resp.Data) > 1 {
		t.Errorf("expected at most 1 item, got %d", len(resp.Data))
	}
	if resp.Pagination.PerPage != 1 {
		t.Errorf("expected per_page 1, got %d", resp.Pagination.PerPage)
	}
}

// Ensure bytes import is used by other tests in package
var _ = bytes.NewReader
