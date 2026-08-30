package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/careeros/api/internal/config"
	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/ingestion"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.LoadIngest()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := ingestion.NewRepository(pool)
	adapters := ingestion.NewAdapterRegistry(ingestion.Credentials{
		USAJobsAPIKey:    cfg.USAJobsAPIKey,
		USAJobsUserAgent: cfg.USAJobsUserAgent,
	}, &http.Client{Timeout: 90 * time.Second})

	service := ingestion.NewService(repo, adapters)
	results, err := service.RunAll(ctx)
	if err != nil {
		slog.Error("ingestion failed", "error", err)
		os.Exit(1)
	}

	failed := 0
	succeeded := 0
	var rawFetched, retained, filteredOut, created, updated, stale, closed int
	for _, result := range results {
		if result.Status == ingestion.RunStatusFailed {
			failed++
			slog.Error("source ingestion failed",
				"source", result.SourceName,
				"error", result.ErrorMessage,
			)
			continue
		}
		succeeded++
		rawFetched += result.RecordsRawFetched
		retained += result.RecordsRetained
		filteredOut += result.RecordsFilteredOut
		created += result.RecordsCreated
		updated += result.RecordsUpdated
		stale += result.RecordsStale
		closed += result.RecordsClosed
		slog.Info("source ingestion succeeded",
			"source", result.SourceName,
			"raw_fetched", result.RecordsRawFetched,
			"retained", result.RecordsRetained,
			"filtered_out", result.RecordsFilteredOut,
			"created", result.RecordsCreated,
			"updated", result.RecordsUpdated,
			"stale", result.RecordsStale,
			"closed", result.RecordsClosed,
		)
	}

	catalog, catalogErr := repo.CountVerifiedCatalog(ctx)
	if catalogErr == nil {
		totalVerified := 0
		byAdapter := map[string]int{}
		for _, item := range catalog {
			byAdapter[item.Adapter] += item.Count
			if item.Count > 0 {
				slog.Info("verified catalog count",
					"source", item.SourceName,
					"adapter", item.Adapter,
					"count", item.Count,
				)
			}
			totalVerified += item.Count
		}
		for adapter, count := range byAdapter {
			if count > 0 {
				slog.Info("verified catalog by provider", "adapter", adapter, "count", count)
			}
		}
		slog.Info("ingestion summary",
			"sources_attempted", len(results),
			"sources_succeeded", succeeded,
			"sources_failed", failed,
			"records_raw_fetched", rawFetched,
			"records_retained", retained,
			"records_filtered_out", filteredOut,
			"records_created", created,
			"records_updated", updated,
			"records_stale", stale,
			"records_closed", closed,
			"verified_catalog_total", totalVerified,
		)
	}

	if failed > 0 {
		os.Exit(1)
	}
}
