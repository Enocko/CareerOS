package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/careeros/api/internal/platform"
)

func TestOpportunitiesBrowseExcludesTestFixtures(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	_ = insertVerifiedOpportunity(t, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities", nil)
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
		title, _ := item["title"].(string)
		if title == "Software Engineer Intern" && item["organization_name"] == "Test Agency" {
			t.Fatal("test fixture leaked into student browse catalog")
		}
	}
}
