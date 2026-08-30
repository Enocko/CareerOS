package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/platform"
	"github.com/careeros/api/internal/server"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestRouterWithAdminPool(t *testing.T, adminEmails []string) (http.Handler, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	cfg := &config.Config{
		DatabaseURL:    databaseURL,
		APIPort:        "8080",
		JWTSecret:      "test-secret-at-least-16-chars",
		JWTExpiryHours: 24,
		CORSOrigin:     "http://localhost:5173",
		AdminEmails:    adminEmails,
		MetricsEnabled: true,
		Environment:    "development",
	}

	return server.NewRouter(cfg, pool), pool
}

func TestResearchVerificationAdminForbiddenForStudent(t *testing.T) {
	router, _ := setupTestRouterWithAdminPool(t, []string{"admin-only@gram.edu"})
	token := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/research/queue", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestResearchVerificationAdminVerifyClosed(t *testing.T) {
	adminEmail := fmt.Sprintf("admin-%d@gram.edu", time.Now().UnixNano())
	router, pool := setupTestRouterWithAdminPool(t, []string{adminEmail})
	adminToken := registerAndGetTokenWithEmail(t, router, adminEmail, "securepass123")
	researchID := insertVerifiedResearchOpportunity(t, pool)

	body, _ := json.Marshal(map[string]any{
		"application_status":      "closed",
		"verification_source_url": "https://example.edu/reu",
		"verification_method":     "manual_official_page",
		"cycle_label":             "Summer 2026",
		"deadline":                "2026-02-01",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/research/opportunities/"+researchID+"/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("verify expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var record map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&record); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if record["application_status"] != "closed" {
		t.Fatalf("expected closed status, got %v", record["application_status"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/"+researchID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail expected 200, got %d", rec.Code)
	}
	var detail map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	meta := detail["type_metadata"].(map[string]any)
	if meta["application_status"] != "closed" {
		t.Fatalf("expected closed in metadata, got %v", meta["application_status"])
	}
}

func TestResearchVerificationRejectsOpenWithoutApplicationURL(t *testing.T) {
	adminEmail := fmt.Sprintf("admin-%d@gram.edu", time.Now().UnixNano())
	router, pool := setupTestRouterWithAdminPool(t, []string{adminEmail})
	adminToken := registerAndGetTokenWithEmail(t, router, adminEmail, "securepass123")
	researchID := insertVerifiedResearchOpportunity(t, pool)

	body, _ := json.Marshal(map[string]any{
		"application_status":      "open",
		"verification_source_url": "https://example.edu/reu",
		"verification_method":     "manual_official_page",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/research/opportunities/"+researchID+"/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestResearchBrowseFiltersByApplicationStatus(t *testing.T) {
	router, pool := setupTestRouterWithAdminPool(t, nil)
	token := registerAndGetToken(t, router)
	openID := insertResearchWithApplicationStatus(t, pool, "open")
	unknownID := insertResearchWithApplicationStatus(t, pool, "unknown")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities?opportunity_type=research&application_status=open", nil)
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
	foundOpen := false
	for _, item := range resp.Data {
		if item["id"] == openID {
			foundOpen = true
		}
		if item["id"] == unknownID {
			t.Fatal("unknown opportunity leaked into open filter")
		}
	}
	if !foundOpen {
		t.Fatal("expected open research opportunity in filtered browse")
	}
}

func registerAndGetTokenWithEmail(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()
	registerBody, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return resp.Token
}

func insertResearchWithApplicationStatus(t *testing.T, pool *pgxpool.Pool, status string) string {
	t.Helper()
	id := uuid.New()
	externalID := fmt.Sprintf("REU-STATUS-%s-%s", status, uuid.NewString())
	now := time.Now().UTC()
	sourceID := "c3000000-0000-4000-8000-000000000002"
	meta := fmt.Sprintf(`{"research_area":"Biology","application_status":"%s","application_status_method":"manual_official_page","availability_verification_method":"manual_official_page"}`, status)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, opportunity_type, work_arrangement, application_url, source_url, source,
			verification_status, verification_method, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count, education_level, type_metadata
		) VALUES (
			$1, $2, $3, $4, 'Status Test University', 'NSF REU Site program description.',
			'research', 'research', 'on_site', NULL, 'https://www.nsf.gov/awardsearch/showAward?AWD_ID=999',
			'U.S. National Science Foundation', 'verified', 'official_source', $5, $5, $5,
			'open', '{}', '{nsf,reu}', 0, 'undergraduate', $6::jsonb
		)
	`, id, sourceID, externalID, fmt.Sprintf("REU Site: Status %s", status), now, meta)
	if err != nil {
		t.Fatalf("insert research opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM research_availability_verifications WHERE opportunity_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}
