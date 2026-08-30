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

	"github.com/careeros/api/internal/applications"
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func insertBrowsableVerifiedOpportunity(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("INT-LIFE-%s", uuid.NewString())
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
			$1, $2, $3, 'Lifecycle Browse Intern', 'Test Agency', 'Browsable lifecycle test opportunity.',
			'internship', 'remote', 'https://www.usajobs.gov/test', 'https://www.usajobs.gov/test', 'USAJobs',
			'verified', $4, $4, $4, 'open', '{}', '{integration}', 0,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{internship,software_engineering}'
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert browsable opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func insertClosedEmploymentOpportunity(t *testing.T, pool *pgxpool.Pool) string {
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
			$1, $2, $3, 'Closed Software Intern', 'Test Agency', 'Closed test opportunity.',
			'internship', 'remote', 'https://www.usajobs.gov/test', 'https://www.usajobs.gov/test', 'USAJobs',
			'closed', $4, $4, $4, 'closed', '{}', '{integration}', 0,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{internship,software_engineering}'
		)
	`, id, sourceID, externalID, now)
	if err != nil {
		t.Fatalf("insert closed opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func TestClosedOpportunityAbsentFromBrowse(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	closedID := insertClosedEmploymentOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}

	var listResp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, item := range listResp.Data {
		if item["id"] == closedID {
			t.Fatal("closed opportunity should not appear in browse list")
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+closedID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("direct access without relationship: expected 404, got %d", rec.Code)
	}
}

func TestClosedOpportunityDirectAccessWhenSaved(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	closedID := insertClosedEmploymentOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/opportunities/"+closedID+"/save", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save closed: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+closedID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail when saved: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail["status"] != "closed" {
		t.Fatalf("expected closed status, got %v", detail["status"])
	}
}

func TestClosedOpportunityDirectAccessAfterPriorView(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	openID := insertBrowsableVerifiedOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+openID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("initial view: expected 200, got %d", rec.Code)
	}

	_, err := pool.Exec(context.Background(), `
		UPDATE opportunities
		SET status = 'closed', verification_status = 'closed', updated_at = now()
		WHERE id = $1
	`, openID)
	if err != nil {
		t.Fatalf("mark closed: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+openID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail after prior view: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestClosedOpportunityExistingApplicationResolves(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	oppID, err := uuid.Parse(insertVerifiedOpportunity(t, pool))
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}

	createBody, _ := json.Marshal(applications.CreateRequest{OpportunityID: oppID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create application: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode application: %v", err)
	}

	_, err = pool.Exec(context.Background(), `
		UPDATE opportunities
		SET status = 'closed', verification_status = $2, updated_at = now()
		WHERE id = $1
	`, oppID, ingestion.VerificationClosed)
	if err != nil {
		t.Fatalf("mark closed: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get application: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+oppID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail with application: expected 200, got %d", rec.Code)
	}
}

func TestClosedOpportunityCannotStartNewApplication(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	closedID := insertClosedEmploymentOpportunity(t, pool)

	oppUUID, err := uuid.Parse(closedID)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}

	createBody, _ := json.Marshal(applications.CreateRequest{OpportunityID: oppUUID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create on closed: expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
