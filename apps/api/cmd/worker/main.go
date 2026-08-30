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
	"github.com/careeros/api/internal/observability"
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
	"github.com/google/uuid"
)

func main() {
	observability.SetDefault("worker", os.Getenv("LOG_LEVEL"))

	cfg, err := config.LoadIngest()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker-" + uuid.NewString()[:8]
	}

	jobCfg := jobs.DefaultConfig()
	jobsRepo := jobs.NewRepository(pool)
	ingestRepo := ingestion.NewRepository(pool)
	notifRepo := notifications.NewRepository(pool)
	adapters := ingestion.NewAdapterRegistry(ingestion.Credentials{
		USAJobsAPIKey:    cfg.USAJobsAPIKey,
		USAJobsUserAgent: cfg.USAJobsUserAgent,
	}, &http.Client{Timeout: 90 * time.Second})
	ingestService := ingestion.NewService(ingestRepo, adapters)
	notifService := notifications.NewService(notifRepo)
	processor := jobs.NewProcessor(jobsRepo, ingestService, ingestRepo, notifService, jobCfg)
	worker := jobs.NewWorker(jobsRepo, processor, jobCfg, workerID)
	worker.Run(ctx)
}
