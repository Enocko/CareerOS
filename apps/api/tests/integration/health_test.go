package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/platform"
	"github.com/careeros/api/internal/server"
)

func TestHealthEndpoint_Liveness(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	defer pool.Close()

	cfg := &config.Config{
		DatabaseURL: databaseURL,
		APIPort:     "8080",
		JWTSecret:   "test-secret-at-least-16-chars",
		CORSOrigin:  "http://localhost:5173",
		MetricsEnabled: true,
		Environment:    "development",
	}

	router := server.NewRouter(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp platform.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
}

func TestReadyEndpoint_Connected(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	defer pool.Close()

	cfg := &config.Config{
		DatabaseURL: databaseURL,
		APIPort:     "8080",
		JWTSecret:   "test-secret-at-least-16-chars",
		CORSOrigin:  "http://localhost:5173",
		MetricsEnabled: true,
		Environment:    "development",
	}

	router := server.NewRouter(cfg, pool)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp platform.ReadinessResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Database != "connected" {
		t.Errorf("expected database connected, got %s", resp.Database)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router, _ := setupTestRouterWithPool(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" || !contains(body, "careeros_http_requests_total") {
		t.Fatalf("expected prometheus metrics body, got: %s", body[:min(200, len(body))])
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
