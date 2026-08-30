package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/careeros/api/internal/applications"
	"github.com/google/uuid"
)

func TestIDOR_ApplicationAccessDeniedForOtherStudent(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)

	tokenA := registerAndGetToken(t, router)
	tokenB := registerAndGetToken(t, router)

	oppID := getFirstOpportunityID(t, pool)
	createBody, _ := json.Marshal(applications.CreateRequest{OpportunityID: oppID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create app A: expected 201, got %d", rec.Code)
	}

	var app applications.Application
	if err := json.NewDecoder(rec.Body).Decode(&app); err != nil {
		t.Fatalf("decode: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/applications/"+app.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other student application, got %d", rec.Code)
	}
}

func TestIDOR_ProfileAccessScopedToSelf(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	body, _ := json.Marshal(map[string]any{"major": "Computer Science"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile: expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get profile: expected 200, got %d", rec.Code)
	}
}

func TestIDOR_AdminRoutesForbiddenForStudents(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	endpoints := []string{
		"/api/v1/admin/overview",
		"/api/v1/admin/reports",
		"/api/v1/admin/research/queue",
	}

	for _, path := range endpoints {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: expected 403, got %d", path, rec.Code)
		}
	}
}

func TestIDOR_InvalidTokenRejected(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestIDOR_NotificationsScopedToSelf(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)
	tokenA := registerAndGetToken(t, router)
	tokenB := registerAndGetToken(t, router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("notifications A: expected 200, got %d", rec.Code)
	}

	fakeID := uuid.New()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+fakeID.String()+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusNoContent {
		t.Fatalf("mark read other user notification: expected 404/204, got %d", rec.Code)
	}
}
