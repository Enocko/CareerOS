package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/notifications"
	"github.com/google/uuid"
)

// IngestionRunner runs ingestion for a source.
type IngestionRunner interface {
	RunSource(ctx context.Context, sourceID uuid.UUID) (ingestion.RunResult, error)
}

// IngestionGuard checks whether a source sync is already active.
type IngestionGuard interface {
	HasRunningIngestion(ctx context.Context, sourceID uuid.UUID) (bool, error)
}

// Processor executes background jobs.
type Processor struct {
	jobsRepo     *Repository
	ingestion    IngestionRunner
	ingestGuard  IngestionGuard
	notifications *notifications.Service
	cfg          Config
}

// NewProcessor creates a job processor.
func NewProcessor(
	jobsRepo *Repository,
	ingestion IngestionRunner,
	ingestGuard IngestionGuard,
	notifService *notifications.Service,
	cfg Config,
) *Processor {
	return &Processor{
		jobsRepo:      jobsRepo,
		ingestion:     ingestion,
		ingestGuard:   ingestGuard,
		notifications: notifService,
		cfg:           cfg,
	}
}

// Process executes one claimed job.
func (p *Processor) Process(ctx context.Context, job *Job, now time.Time) error {
	switch job.JobType {
	case TypeIngestSource:
		return p.processIngest(ctx, job, now)
	case TypeDeadlineReminder:
		return p.processReminder(ctx, job, now)
	default:
		return fmt.Errorf("unknown job type %q", job.JobType)
	}
}

func (p *Processor) processIngest(ctx context.Context, job *Job, now time.Time) error {
	var payload IngestSourcePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode ingest payload: %w", err)
	}

	running, err := p.ingestGuard.HasRunningIngestion(ctx, payload.SourceID)
	if err != nil {
		return err
	}
	if running {
		slog.Info("ingest job skipped overlapping sync", "job_id", job.ID, "source_id", payload.SourceID)
		return nil
	}

	slog.Info("scheduled ingestion started", "job_id", job.ID, "source_id", payload.SourceID)
	result, runErr := p.ingestion.RunSource(ctx, payload.SourceID)
	if runErr != nil {
		return runErr
	}
	slog.Info("scheduled ingestion completed",
		"job_id", job.ID,
		"source_id", payload.SourceID,
		"status", result.Status,
		"retained", result.RecordsRetained,
	)
	_ = now
	return nil
}

func (p *Processor) processReminder(ctx context.Context, job *Job, now time.Time) error {
	var payload DeadlineReminderPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode reminder payload: %w", err)
	}

	idempotencyKey := ReminderIdempotencyKey(
		payload.UserID, payload.OpportunityID, payload.ApplicationID,
		payload.DeadlineDate, payload.WindowDays,
	)

	created, err := p.notifications.CreateDeadlineReminder(ctx, notifications.DeadlineReminderInput{
		UserID:           payload.UserID,
		OpportunityID:    payload.OpportunityID,
		ApplicationID:    payload.ApplicationID,
		ReminderKind:     payload.ReminderKind,
		WindowDays:       payload.WindowDays,
		OpportunityTitle: payload.OpportunityTitle,
		DeadlineDate:     payload.DeadlineDate,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		return err
	}
	if created {
		slog.Info("reminder created", "job_id", job.ID, "user_id", payload.UserID, "opportunity_id", payload.OpportunityID, "window_days", payload.WindowDays)
	} else {
		slog.Info("duplicate reminder suppressed", "job_id", job.ID, "idempotency_key", idempotencyKey)
	}
	_ = now
	return nil
}
