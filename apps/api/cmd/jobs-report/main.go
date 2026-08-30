package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
)

func main() {
	cfg, err := config.LoadIngest()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	now := time.Now().UTC()
	jobsRepo := jobs.NewRepository(pool)
	ingestRepo := ingestion.NewRepository(pool)
	notifRepo := notifications.NewRepository(pool)

	metrics, err := jobsRepo.Metrics(ctx, now)
	if err != nil {
		log.Fatalf("job metrics: %v", err)
	}
	notifCount, _ := notifRepo.CountCreated(ctx)
	lastIngest, _ := ingestRepo.LastSuccessfulIngestionBySource(ctx)

	fmt.Println("=== CareerOS Background Jobs Report ===")
	fmt.Printf("Queued: %d\n", metrics.Queued)
	fmt.Printf("Processing: %d\n", metrics.Processing)
	fmt.Printf("Retryable: %d\n", metrics.Retryable)
	fmt.Printf("Failed: %d\n", metrics.Failed)
	fmt.Printf("Completed: %d\n", metrics.Completed)
	fmt.Printf("Total retry attempts: %d\n", metrics.TotalRetries)
	if metrics.OldestQueuedAge > 0 {
		fmt.Printf("Oldest queued job age: %s\n", metrics.OldestQueuedAge.Round(time.Second))
	}
	fmt.Printf("Notifications created: %d\n", notifCount)
	fmt.Println("\nLast successful ingestion by source:")
	for name, ts := range lastIngest {
		fmt.Printf("  %s: %s\n", name, ts.Format(time.RFC3339))
	}
}
