package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpportunityReportStudentFlow(t *testing.T) {
	adminEmail := fmt.Sprintf("admin-report-%d@gram.edu", time.Now().UnixNano())
	router, pool := setupTestRouterWithAdminPool(t, []string{adminEmail})
	studentToken := registerAndGetToken(t, router)
	adminToken := registerAndGetTokenWithEmail(t, router, adminEmail, "securepass123")
	oppID := insertBrowsableEmploymentOpportunity(t, pool)

	body, _ := json.Marshal(map[string]any{
		"reason": "broken_link",
		"note":   "Link returns 404",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/opportunities/"+oppID+"/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+studentToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminToken)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 admin list, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	patchBody, _ := json.Marshal(map[string]string{"status": "resolved"})
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/reports/opportunities/"+oppID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+adminToken)
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200 resolve, got %d body=%s", patchRec.Code, patchRec.Body.String())
	}
}

func TestOpportunityReportForbiddenForNonAdminQueue(t *testing.T) {
	router, _ := setupTestRouterWithAdminPool(t, []string{"admin-only@gram.edu"})
	token := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
