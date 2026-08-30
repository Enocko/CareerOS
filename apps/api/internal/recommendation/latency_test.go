package recommendation_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/profile"
	"github.com/careeros/api/internal/recommendation"
	"github.com/google/uuid"
)

func TestRecommendationLatency(t *testing.T) {
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

	recRepo := recommendation.NewRepository(pool)
	profileRepo := profile.NewRepository(pool)
	service := recommendation.NewService(recRepo, profileRepo)

	studentID := uuid.New() // no profile; cold-start path
	start := time.Now()
	_, meta, err := service.Recommend(ctx, studentID, recommendation.ListFilter{Page: 1, PerPage: 20})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("recommend: %v", err)
	}

	t.Logf("catalog_scored=%d eligible=%d latency_ms=%d", meta.CatalogScored, meta.EligibleCount, elapsed.Milliseconds())
	if elapsed > 2*time.Second {
		t.Fatalf("recommendation latency too high: %v", elapsed)
	}
	_ = profileRepo
}
