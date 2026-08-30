package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/notifications"
)

// Scheduler enqueues background work.
type Scheduler struct {
	jobsRepo  *Repository
	ingestRepo *ingestion.Repository
	notifRepo *notifications.Repository
	cfg       Config
}

// NewScheduler creates a scheduler.
func NewScheduler(jobsRepo *Repository, ingestRepo *ingestion.Repository, notifRepo *notifications.Repository, cfg Config) *Scheduler {
	return &Scheduler{jobsRepo: jobsRepo, ingestRepo: ingestRepo, notifRepo: notifRepo, cfg: cfg}
}

// EnqueueDueIngestion enqueues ingest jobs for sources past their sync interval.
func (s *Scheduler) EnqueueDueIngestion(ctx context.Context, now time.Time) (int, error) {
	sources, err := s.ingestRepo.ListDueSources(ctx, now)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, source := range sources {
		running, err := s.ingestRepo.HasRunningIngestion(ctx, source.ID)
		if err != nil {
			return enqueued, err
		}
		if running {
			slog.Info("ingest enqueue skipped running source", "source_id", source.ID, "source_name", source.Name)
			continue
		}
		active, err := s.jobsRepo.HasActiveIngestJob(ctx, source.ID)
		if err != nil {
			return enqueued, err
		}
		if active {
			slog.Info("ingest enqueue skipped active job", "source_id", source.ID, "source_name", source.Name)
			continue
		}

		id, created, err := s.jobsRepo.Enqueue(ctx, TypeIngestSource, IngestSourcePayload{SourceID: source.ID},
			IngestIdempotencyKey(source.ID), now, s.cfg.MaxAttempts)
		if err != nil {
			return enqueued, err
		}
		if created {
			enqueued++
			slog.Info("job created", "job_id", id, "job_type", TypeIngestSource, "source_id", source.ID)
		}
	}
	return enqueued, nil
}

// EnqueueDeadlineReminders enqueues reminder jobs for upcoming deadlines.
func (s *Scheduler) EnqueueDeadlineReminders(ctx context.Context, now time.Time) (int, error) {
	enqueued := 0
	for _, window := range s.cfg.ReminderWindows {
		targetDate := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, window)
		deadlineStr := targetDate.Format("2006-01-02")

		appRows, err := s.notifRepo.ListApplicationDeadlineCandidates(ctx, targetDate)
		if err != nil {
			return enqueued, err
		}
		for _, row := range appRows {
			appID := row.ApplicationID
			id, created, err := s.jobsRepo.Enqueue(ctx, TypeDeadlineReminder, DeadlineReminderPayload{
				UserID:           row.UserID,
				OpportunityID:    row.OpportunityID,
				ApplicationID:    &appID,
				ReminderKind:     "application",
				DeadlineDate:     deadlineStr,
				WindowDays:       window,
				OpportunityTitle: row.Title,
			}, ReminderIdempotencyKey(row.UserID, row.OpportunityID, &appID, deadlineStr, window), now, s.cfg.MaxAttempts)
			if err != nil {
				return enqueued, err
			}
			if created {
				enqueued++
				slog.Info("job created", "job_id", id, "job_type", TypeDeadlineReminder, "window_days", window, "kind", "application")
			}
		}

		savedRows, err := s.notifRepo.ListSavedDeadlineCandidates(ctx, targetDate)
		if err != nil {
			return enqueued, err
		}
		for _, row := range savedRows {
			id, created, err := s.jobsRepo.Enqueue(ctx, TypeDeadlineReminder, DeadlineReminderPayload{
				UserID:           row.UserID,
				OpportunityID:    row.OpportunityID,
				ReminderKind:     "saved",
				DeadlineDate:     deadlineStr,
				WindowDays:       window,
				OpportunityTitle: row.Title,
			}, ReminderIdempotencyKey(row.UserID, row.OpportunityID, nil, deadlineStr, window), now, s.cfg.MaxAttempts)
			if err != nil {
				return enqueued, err
			}
			if created {
				enqueued++
				slog.Info("job created", "job_id", id, "job_type", TypeDeadlineReminder, "window_days", window, "kind", "saved")
			}
		}
	}
	return enqueued, nil
}

// RunOnce executes one scheduler tick.
func (s *Scheduler) RunOnce(ctx context.Context, now time.Time) error {
	ingestCount, err := s.EnqueueDueIngestion(ctx, now)
	if err != nil {
		return fmt.Errorf("enqueue ingestion: %w", err)
	}
	reminderCount, err := s.EnqueueDeadlineReminders(ctx, now)
	if err != nil {
		return fmt.Errorf("enqueue reminders: %w", err)
	}
	slog.Info("scheduler tick completed", "ingest_jobs_enqueued", ingestCount, "reminder_jobs_enqueued", reminderCount)
	return nil
}
