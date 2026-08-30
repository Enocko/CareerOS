package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/careeros/api/internal/applications"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getFirstOpportunityID(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	oppID, err := uuid.Parse(insertVerifiedOpportunity(t, pool))
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	return oppID
}

func TestSavedOpportunitiesFlow(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	oppID := getFirstOpportunityID(t, pool)

	// Save
	req := httptest.NewRequest(http.MethodPost, "/api/v1/opportunities/"+oppID.String()+"/save", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("save: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Idempotent save
	req = httptest.NewRequest(http.MethodPost, "/api/v1/opportunities/"+oppID.String()+"/save", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("save again: expected 200, got %d", rec.Code)
	}

	// List saved
	req = httptest.NewRequest(http.MethodGet, "/api/v1/saved-opportunities", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list saved: expected 200, got %d", rec.Code)
	}

	var savedResp platform.PaginatedResponse[map[string]any]
	if err := json.NewDecoder(rec.Body).Decode(&savedResp); err != nil {
		t.Fatalf("decode saved: %v", err)
	}
	if savedResp.Pagination.Total < 1 {
		t.Error("expected at least 1 saved opportunity")
	}

	// Unsave
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/opportunities/"+oppID.String()+"/save", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unsave: expected 204, got %d", rec.Code)
	}
}

func TestApplicationsFlow(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	oppID := getFirstOpportunityID(t, pool)

	// Create application
	createBody, _ := json.Marshal(applications.CreateRequest{
		OpportunityID: oppID,
		Notes:         strPtr("Found via CareerOS seed data"),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.CurrentStatus != "saved" {
		t.Errorf("expected status saved, got %s", created.CurrentStatus)
	}

	// Duplicate
	req = httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d", rec.Code)
	}

	// Update status to applied
	updateBody, _ := json.Marshal(applications.UpdateRequest{
		CurrentStatus: strPtr("applied"),
		Notes:         strPtr("Submitted via company portal"),
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/applications/"+created.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var updated applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.CurrentStatus != "applied" {
		t.Errorf("expected status applied, got %s", updated.CurrentStatus)
	}
	if updated.DateApplied == nil {
		t.Error("expected date_applied to be set")
	}

	// Dashboard list
	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}

	// Detail with history
	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("detail: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var detail applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.StatusHistory) < 2 {
		t.Errorf("expected at least 2 history entries, got %d", len(detail.StatusHistory))
	}
}

func TestRemoveApplicationTracking(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	oppID := getFirstOpportunityID(t, pool)

	createBody, _ := json.Marshal(applications.CreateRequest{OpportunityID: oppID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", rec.Code)
	}

	var created applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/applications/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("after delete: expected 404, got %d", rec.Code)
	}
}

func TestApplicationRejectsNonEmploymentOpportunity(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	oppID := uuid.New()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO opportunities (
			id, title, organization_name, description, category, opportunity_type,
			work_arrangement, application_url, source, status, skills, tags,
			verification_status, type_metadata
		) VALUES (
			$1, 'UNCF STEM Scholars Program', 'UNCF', 'Scholarship program.',
			'scholarship', 'scholarship', 'remote', 'https://scholarships.uncf.org', 'manual', 'open',
			'{}', '{}', 'unverified', '{}'::jsonb
		)
	`, oppID)
	if err != nil {
		t.Fatalf("insert scholarship opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(t.Context(), `DELETE FROM opportunities WHERE id = $1`, oppID)
	})

	createBody, _ := json.Marshal(applications.CreateRequest{OpportunityID: oppID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-employment application, got %d body=%s", rec.Code, rec.Body.String())
	}
}
