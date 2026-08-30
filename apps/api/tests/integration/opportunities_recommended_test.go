package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecommendedOpportunitiesRankByProfile(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)

	upsertProfile(t, router, token, map[string]any{
		"major":            "Computer Science",
		"career_interests": []string{"software engineering"},
		"skills":           []string{"Python"},
		"experience_level": "intern",
		"work_arrangement": "remote",
	})

	sweID := insertRecommendableOpportunity(t, pool, "REC-SWE-"+uuid.NewString()[:8],
		"Software Engineer Intern", "software_engineering", "internship", "remote")
	otherID := insertRecommendableOpportunity(t, pool, "REC-OTHER-"+uuid.NewString()[:8],
		"Technology Analyst Intern", "other_technical", "internship", "on_site")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/recommended?per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []struct {
			MatchScore  int `json:"match_score"`
			Opportunity struct {
				ID string `json:"id"`
			} `json:"opportunity"`
		} `json:"data"`
		Meta struct {
			EligibleCount int `json:"eligible_count"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Meta.EligibleCount < 2 {
		t.Fatalf("expected at least 2 eligible recommendations, got %d", resp.Meta.EligibleCount)
	}

	var sweScore, otherScore int
	foundSWE, foundOther := false, false
	for _, item := range resp.Data {
		switch item.Opportunity.ID {
		case sweID:
			sweScore = item.MatchScore
			foundSWE = true
		case otherID:
			otherScore = item.MatchScore
			foundOther = true
		}
	}
	if !foundSWE || !foundOther {
		t.Fatalf("expected both fixture opportunities in response (swe=%v other=%v)", foundSWE, foundOther)
	}
	if sweScore <= otherScore {
		t.Fatalf("expected SWE (%d) to outrank other technical (%d)", sweScore, otherScore)
	}
}

func TestRecommendedExcludesAppliedOpportunity(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	userID := getUserIDFromToken(t, router, token)

	oppID := insertRecommendableOpportunity(t, pool, "REC-APPLIED-"+uuid.NewString()[:8],
		"Applied SWE Intern", "software_engineering", "internship", "remote")

	_, err := pool.Exec(context.Background(), `
		INSERT INTO applications (student_id, opportunity_id, current_status)
		VALUES ($1, $2, 'applied')
	`, userID, oppID)
	if err != nil {
		t.Fatalf("insert application: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM applications WHERE opportunity_id = $1`, oppID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/opportunities/recommended?per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data []struct {
			Opportunity struct {
				ID string `json:"id"`
			} `json:"opportunity"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	for _, item := range resp.Data {
		if item.Opportunity.ID == oppID {
			t.Fatal("applied opportunity should not appear in recommendations")
		}
	}
}

func insertRecommendableOpportunity(t *testing.T, pool *pgxpool.Pool, externalID, title, family, expLevel, work string) string {
	t.Helper()
	id := uuid.New()
	sourceID := "c3000000-0000-4000-8000-000000000001"
	now := time.Now().UTC()
	deadline := now.Add(30 * 24 * time.Hour)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count, deadline,
			experience_level, career_family, education_level, relevance_tier, classification_reasons
		) VALUES (
			$1, $2, $3, $4, 'Rec Test Co', 'Recommendation fixture.',
			'internship', $5, 'https://example.com', 'https://example.com', 'Fixture',
			'verified', $6, $6, $6, 'open', '{Python}', '{}', 0, $7,
			$8, $9, 'unspecified', 'high_confidence_technical', '{fixture}'
		)
	`, id, sourceID, externalID, title, work, now, deadline, expLevel, family)
	if err != nil {
		t.Fatalf("insert recommendable opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id.String()
}

func upsertProfile(t *testing.T, router http.Handler, token string, body map[string]any) {
	t.Helper()
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upsert profile: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func getUserIDFromToken(t *testing.T, router http.Handler, token string) uuid.UUID {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	id, err := uuid.Parse(resp.ID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	return id
}
