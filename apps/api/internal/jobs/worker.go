package jobs

import (
	"context"
	"log/slog"
	"time"
)

// Worker claims and processes background jobs.
type Worker struct {
	repo      *Repository
	processor *Processor
	cfg       Config
	workerID  string
}

// NewWorker creates a background worker.
func NewWorker(repo *Repository, processor *Processor, cfg Config, workerID string) *Worker {
	return &Worker{repo: repo, processor: processor, cfg: cfg, workerID: workerID}
}

// Run polls and processes jobs until context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("worker started", "worker_id", w.workerID)
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped", "worker_id", w.workerID)
			return
		default:
			w.processOne(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.cfg.WorkerPoll):
			}
		}
	}
}

func (w *Worker) processOne(ctx context.Context) {
	now := time.Now().UTC()
	if reclaimed, err := w.repo.ReclaimStaleProcessing(ctx, now, w.cfg.ProcessingLease); err != nil {
		slog.Error("stale job reclaim failed", "worker_id", w.workerID, "error", err)
	} else if reclaimed > 0 {
		slog.Warn("stale processing jobs reclaimed", "worker_id", w.workerID, "count", reclaimed, "outcome", "reclaimed")
	}

	job, err := w.repo.ClaimNext(ctx, w.workerID, now)
	if err != nil {
		slog.Error("job claim failed", "worker_id", w.workerID, "error", err)
		return
	}
	if job == nil {
		return
	}

	slog.Info("job claimed", "job_id", job.ID, "job_type", job.JobType, "worker_id", w.workerID, "attempt", job.Attempts)

	procErr := w.processor.Process(ctx, job, now)
	if procErr == nil {
		if err := w.repo.MarkCompleted(ctx, job.ID, now); err != nil {
			slog.Error("job complete persist failed", "job_id", job.ID, "error", err)
			return
		}
		slog.Info("job completed", "job_id", job.ID, "job_type", job.JobType, "outcome", "completed")
		return
	}

	backoff := w.cfg.NextBackoff(job.Attempts)
	runAt := now.Add(backoff)
	permanent, err := w.repo.MarkRetryable(ctx, job.ID, job.Attempts, job.MaxAttempts, procErr.Error(), runAt, now)
	if err != nil {
		slog.Error("job retry persist failed", "job_id", job.ID, "error", err)
		return
	}
	if permanent {
		slog.Error("job permanently failed", "job_id", job.ID, "job_type", job.JobType, "attempts", job.Attempts, "error", procErr.Error(), "outcome", "failed")
	} else {
		slog.Info("job retry scheduled", "job_id", job.ID, "attempt", job.Attempts, "run_at", runAt, "backoff_ms", backoff.Milliseconds())
	}
}

// ProcessOne runs a single claim/process cycle (for tests).
func (w *Worker) ProcessOne(ctx context.Context) {
	w.processOne(ctx)
}
