package config

import (
	"strings"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable")
	t.Setenv("JWT_SECRET", "test-secret-at-least-16-chars")
	t.Setenv("API_PORT", "8080")
	t.Setenv("LOG_LEVEL", "info")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.APIPort != "8080" {
		t.Errorf("expected APIPort 8080, got %s", cfg.APIPort)
	}
	if cfg.JWTExpiryHours != 24 {
		t.Errorf("expected JWTExpiryHours 24, got %d", cfg.JWTExpiryHours)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "test-secret-at-least-16-chars")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/careeros")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/careeros")
	t.Setenv("JWT_SECRET", "short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 16") {
		t.Fatalf("expected short JWT_SECRET error, got %v", err)
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/careeros")
	t.Setenv("JWT_SECRET", "test-secret-at-least-16-chars")
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("expected LOG_LEVEL error, got %v", err)
	}
}
