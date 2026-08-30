package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/profile"
)

func registerAndGetToken(t *testing.T, router http.Handler) string {
	t.Helper()

	email := uniqueEmail(t)
	password := "securepass123"

	body, _ := json.Marshal(auth.RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp auth.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	return resp.Token
}

func TestProfileGetPutUpsert(t *testing.T) {
	router := setupTestRouter(t)
	token := registerAndGetToken(t, router)

	// GET before profile exists → 404
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("get before create: expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}

	// PUT creates profile
	year := 2027
	updateBody, _ := json.Marshal(profile.UpdateRequest{
		FirstName:       strPtr("Jordan"),
		LastName:        strPtr("Smith"),
		Major:           strPtr("Computer Science"),
		GraduationYear:  &year,
		Skills:          []string{"Python", "Go"},
		WorkArrangement: strPtr("remote"),
		ExperienceLevel: strPtr("intern"),
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("put create: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var created profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if created.FirstName == nil || *created.FirstName != "Jordan" {
		t.Errorf("expected first name Jordan, got %v", created.FirstName)
	}
	if created.University == nil || *created.University != "Grambling State University" {
		t.Errorf("expected default university, got %v", created.University)
	}

	// GET returns profile
	req = httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get after create: expected 200, got %d", rec.Code)
	}

	// PUT updates profile
	updatedBody, _ := json.Marshal(profile.UpdateRequest{
		FirstName: strPtr("Jordan"),
		LastName:  strPtr("Williams"),
		Major:     strPtr("Computer Science"),
		Skills:    []string{"Python", "Go", "SQL"},
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(updatedBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("put update: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var updated profile.Profile
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	if updated.LastName == nil || *updated.LastName != "Williams" {
		t.Errorf("expected last name Williams, got %v", updated.LastName)
	}
	if len(updated.Skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(updated.Skills))
	}
}

func TestProfileUnauthorized(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestProfileValidation(t *testing.T) {
	router := setupTestRouter(t)
	token := registerAndGetToken(t, router)

	year := 2010
	body, _ := json.Marshal(profile.UpdateRequest{GraduationYear: &year})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func strPtr(v string) *string { return &v }
