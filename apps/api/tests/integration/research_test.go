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

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestResearchOpportunityBrowseFilter(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	empID := insertVerifiedEmploymentOpportunity(t, pool)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?opportunity_type=research", nil)
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

	foundResearch := false
	for _, item := range resp.Data {
		id, _ := item["id"].(string)
		if id == researchID {
			foundResearch = true
		}
		if id == empID {
			t.Fatal("employment opportunity leaked into research browse")
		}
		if oppType, _ := item["opportunity_type"].(string); oppType != "research" {
			t.Fatalf("expected research type, got %q", oppType)
		}
	}
	if !foundResearch {
		t.Fatal("expected verified research opportunity in research browse")
	}
}

func TestResearchOpportunityDetailAndSave(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+researchID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail expected 200, got %d", rec.Code)
	}

	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail["opportunity_type"] != "research" {
		t.Fatalf("expected research detail, got %v", detail["opportunity_type"])
	}

	saveReq := httptest.NewRequest(http.MethodPost, "/api/v1/opportunities/"+researchID+"/save", nil)
	saveReq.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, saveReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("save expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+researchID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail after save: %v", err)
	}
	if saved, _ := detail["is_saved"].(bool); !saved {
		t.Fatal("expected research opportunity to remain saved after reload")
	}
}

func TestResearchOpportunityShowsUnknownApplicationStatus(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+researchID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail expected 200, got %d", rec.Code)
	}

	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	meta, ok := detail["type_metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected type_metadata object")
	}
	if meta["application_status"] != "unknown" {
		t.Fatalf("expected unknown application_status, got %v", meta["application_status"])
	}
	if detail["application_url"] != nil {
		t.Fatalf("expected no application_url, got %v", detail["application_url"])
	}
}

func TestResearchApplicationRejected(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	researchID := insertVerifiedResearchOpportunity(t, pool)

	createBody, _ := json.Marshal(map[string]string{"opportunity_id": researchID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for research application, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func insertVerifiedEmploymentOpportunity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("API-TEST-%s", uuid.NewString())
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
			$1, $2, $3, 'Software Engineer Intern', 'Test Agency', 'Verified test opportunity.',
			'internship', 'employment', 'remote', 'https://www.usajobs.gov/test', 'https://www.usajobs.gov/test', 'USAJobs',
			'verified', $4, $4, $4, 'open', '{}', '{integration}', 0,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{internship,software_engineering}',
			'{}'::jsonb, 'official_source'
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert employment opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func insertVerifiedResearchOpportunity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("REU-TEST-%s", uuid.NewString())
	now := time.Now().UTC()
	sourceID := "c3000000-0000-4000-8000-000000000002"
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, opportunity_type, work_arrangement, application_url, source_url, source,
			verification_status, verification_method, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count, education_level, type_metadata
		) VALUES (
			$1, $2, $3, 'REU Site: Test Quantum Research', 'Test University', 'NSF REU Site program description.',
			'research', 'research', 'on_site', NULL, 'https://www.nsf.gov/awardsearch/showAward?AWD_ID=123',
			'U.S. National Science Foundation', 'verified', 'official_source', $4, $4, $4,
			'open', '{}', '{nsf,reu}', 0, 'undergraduate',
			'{"research_area":"Quantum Computing","duration_weeks":10,"application_status":"unknown","application_status_method":"nsf_award_only","availability_verification_method":"unknown"}'::jsonb
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert research opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}
