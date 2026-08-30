package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_FromRepoRoot(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}

	envPath := filepath.Join(root, ".env")
	if _, err := os.Stat(envPath); err != nil {
		t.Skip(".env not present at repo root")
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Chdir(filepath.Join(root, "apps", "api"))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load from .env, got error: %v", err)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("expected DATABASE_URL to be loaded from .env")
	}
	if cfg.JWTSecret == "" {
		t.Fatal("expected JWT_SECRET to be loaded from .env")
	}
}
