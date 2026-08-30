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

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestRouter(t *testing.T) http.Handler {
	router, _ := setupTestRouterWithPool(t)
	return router
}

func setupTestRouterWithPool(t *testing.T) (http.Handler, *pgxpool.Pool) {
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
		Environment:    "development",
		MetricsEnabled: true,
		CookieSecure:   false,
		CookieSameSite: "Lax",
	}

	return server.NewRouter(cfg, pool), pool
}

func uniqueEmail(t *testing.T) string {
	return fmt.Sprintf("test-%s-%d@gram.edu", t.Name(), time.Now().UnixNano())
}

func TestAuthRegisterLoginMe(t *testing.T) {
	router := setupTestRouter(t)
	email := uniqueEmail(t)
	password := "securepass123"

	// Register
	registerBody, _ := json.Marshal(auth.RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var registerResp auth.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&registerResp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResp.Token == "" {
		t.Fatal("expected token in register response")
	}
	if registerResp.User.Email != auth.NormalizeEmail(email) {
		t.Errorf("expected email %s, got %s", auth.NormalizeEmail(email), registerResp.User.Email)
	}

	// Login
	loginBody, _ := json.Marshal(auth.LoginRequest{Email: email, Password: password})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var loginResp auth.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("expected token in login response")
	}

	// Me
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var meResp auth.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&meResp); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meResp.Email != auth.NormalizeEmail(email) {
		t.Errorf("expected email %s, got %s", auth.NormalizeEmail(email), meResp.Email)
	}
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	router := setupTestRouter(t)
	email := uniqueEmail(t)
	password := "securepass123"

	body, _ := json.Marshal(auth.RegisterRequest{Email: email, Password: password})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("first register: expected 201, got %d", rec.Code)
		}
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("duplicate register: expected 409, got %d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	router := setupTestRouter(t)

	body, _ := json.Marshal(auth.LoginRequest{Email: "nonexistent@gram.edu", Password: "wrongpass1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthMeUnauthorized(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	router := setupTestRouter(t)

	body, _ := json.Marshal(auth.RegisterRequest{Email: "bad", Password: "short"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
