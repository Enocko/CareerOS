package main

import (
	"context"
	"flag"
	"log/slog"
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
)

func main() {
	once := flag.Bool("once", false, "run one scheduler tick and exit")
	flag.Parse()

	observability.SetDefault("scheduler", os.Getenv("LOG_LEVEL"))

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

	jobCfg := jobs.DefaultConfig()
	jobsRepo := jobs.NewRepository(pool)
	ingestRepo := ingestion.NewRepository(pool)
	notifRepo := notifications.NewRepository(pool)
	scheduler := jobs.NewScheduler(jobsRepo, ingestRepo, notifRepo, jobCfg)

	if *once {
		if err := scheduler.RunOnce(ctx, time.Now().UTC()); err != nil {
			slog.Error("scheduler tick failed", "error", err)
			os.Exit(1)
		}
		return
	}

	ticker := time.NewTicker(jobCfg.SchedulerTick)
	defer ticker.Stop()
	slog.Info("scheduler started", "tick_seconds", jobCfg.SchedulerTick.Seconds())

	if err := scheduler.RunOnce(ctx, time.Now().UTC()); err != nil {
		slog.Error("scheduler tick failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			if err := scheduler.RunOnce(ctx, time.Now().UTC()); err != nil {
				slog.Error("scheduler tick failed", "error", err)
			}
		}
	}
}
