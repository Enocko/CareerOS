package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists background jobs.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a jobs repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Enqueue inserts a job unless an active duplicate idempotency key exists.
func (r *Repository) Enqueue(ctx context.Context, jobType string, payload any, idempotencyKey string, runAt time.Time, maxAttempts int) (uuid.UUID, bool, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("marshal job payload: %w", err)
	}
	if maxAttempts <= 0 {
		maxAttempts = DefaultConfig().MaxAttempts
	}

	var id uuid.UUID
	err = r.pool.QueryRow(ctx, `
		INSERT INTO background_jobs (job_type, payload, idempotency_key, run_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) WHERE status IN ('queued', 'processing', 'retryable')
		DO NOTHING
		RETURNING id
	`, jobType, raw, idempotencyKey, runAt, maxAttempts).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, fmt.Errorf("enqueue job: %w", err)
	}
	return id, true, nil
}

// ClaimNext atomically claims one due job for a worker.
func (r *Repository) ClaimNext(ctx context.Context, workerID string, now time.Time) (*Job, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var job Job
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id, job_type, payload, status, idempotency_key, attempts, max_attempts,
		       run_at, locked_at, locked_by, last_error, created_at, updated_at, completed_at
		FROM background_jobs
		WHERE status IN ($1, $2) AND run_at <= $3
		ORDER BY run_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, StatusQueued, StatusRetryable, now).Scan(
		&job.ID, &job.JobType, &payload, &job.Status, &job.IdempotencyKey,
		&job.Attempts, &job.MaxAttempts, &job.RunAt, &job.LockedAt, &job.LockedBy,
		&job.LastError, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select claimable job: %w", err)
	}
	job.Payload = payload

	_, err = tx.Exec(ctx, `
		UPDATE background_jobs
		SET status = $2, locked_at = $3, locked_by = $4, attempts = attempts + 1, updated_at = $3
		WHERE id = $1
	`, job.ID, StatusProcessing, now, workerID)
	if err != nil {
		return nil, fmt.Errorf("mark job processing: %w", err)
	}
	job.Status = StatusProcessing
	job.Attempts++
	locked := now
	job.LockedAt = &locked
	job.LockedBy = &workerID

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim tx: %w", err)
	}
	return &job, nil
}

// ClaimByID atomically claims a specific due job for a worker.
func (r *Repository) ClaimByID(ctx context.Context, workerID string, jobID uuid.UUID, now time.Time) (*Job, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var job Job
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id, job_type, payload, status, idempotency_key, attempts, max_attempts,
		       run_at, locked_at, locked_by, last_error, created_at, updated_at, completed_at
		FROM background_jobs
		WHERE id = $1 AND status IN ($2, $3) AND run_at <= $4
		FOR UPDATE
	`, jobID, StatusQueued, StatusRetryable, now).Scan(
		&job.ID, &job.JobType, &payload, &job.Status, &job.IdempotencyKey,
		&job.Attempts, &job.MaxAttempts, &job.RunAt, &job.LockedAt, &job.LockedBy,
		&job.LastError, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select claimable job: %w", err)
	}
	job.Payload = payload

	_, err = tx.Exec(ctx, `
		UPDATE background_jobs
		SET status = $2, locked_at = $3, locked_by = $4, attempts = attempts + 1, updated_at = $3
		WHERE id = $1
	`, job.ID, StatusProcessing, now, workerID)
	if err != nil {
		return nil, fmt.Errorf("mark job processing: %w", err)
	}
	job.Status = StatusProcessing
	job.Attempts++
	locked := now
	job.LockedAt = &locked
	job.LockedBy = &workerID

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim tx: %w", err)
	}
	return &job, nil
}

// MarkCompleted marks a job successful.
func (r *Repository) MarkCompleted(ctx context.Context, jobID uuid.UUID, now time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = $2, completed_at = $3, updated_at = $3, locked_at = NULL, locked_by = NULL, last_error = NULL
		WHERE id = $1
	`, jobID, StatusCompleted, now)
	if err != nil {
		return fmt.Errorf("mark job completed: %w", err)
	}
	return nil
}

// MarkRetryable schedules a retry or permanent failure.
func (r *Repository) MarkRetryable(ctx context.Context, jobID uuid.UUID, attempts, maxAttempts int, errMsg string, runAt time.Time, now time.Time) (permanent bool, err error) {
	status := StatusRetryable
	if attempts >= maxAttempts {
		status = StatusFailed
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = $2, run_at = $3, last_error = $4, updated_at = $5, locked_at = NULL, locked_by = NULL
		WHERE id = $1
	`, jobID, status, runAt, errMsg, now)
	if err != nil {
		return false, fmt.Errorf("mark job retryable: %w", err)
	}
	return status == StatusFailed, nil
}

// ReclaimStaleProcessing moves abandoned processing jobs back to retryable.
func (r *Repository) ReclaimStaleProcessing(ctx context.Context, now time.Time, lease time.Duration) (int, error) {
	if lease <= 0 {
		lease = DefaultConfig().ProcessingLease
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE background_jobs
		SET status = $1,
		    run_at = $2,
		    updated_at = $2,
		    locked_at = NULL,
		    locked_by = NULL,
		    last_error = COALESCE(last_error, '') || ' [reclaimed stale processing lease]'
		WHERE status = $3
		  AND locked_at IS NOT NULL
		  AND locked_at < $4
	`, StatusRetryable, now, StatusProcessing, now.Add(-lease))
	if err != nil {
		return 0, fmt.Errorf("reclaim stale processing jobs: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// HasActiveIngestJob reports whether a source already has a pending ingest job.
func (r *Repository) HasActiveIngestJob(ctx context.Context, sourceID uuid.UUID) (bool, error) {
	key := IngestIdempotencyKey(sourceID)
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM background_jobs
			WHERE idempotency_key = $1
			  AND status IN ($2, $3, $4)
		)
	`, key, StatusQueued, StatusProcessing, StatusRetryable).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has active ingest job: %w", err)
	}
	return exists, nil
}

// IngestIdempotencyKey returns the active ingest dedupe key for a source.
func IngestIdempotencyKey(sourceID uuid.UUID) string {
	return fmt.Sprintf("ingest:source:%s", sourceID)
}

// ReminderIdempotencyKey builds a reminder dedupe key.
func ReminderIdempotencyKey(userID, opportunityID uuid.UUID, applicationID *uuid.UUID, deadline string, windowDays int) string {
	appPart := "saved"
	if applicationID != nil {
		appPart = applicationID.String()
	}
	return fmt.Sprintf("reminder:%s:%s:%s:%s:%dd", userID, opportunityID, appPart, deadline, windowDays)
}

// Metrics returns queue statistics.
func (r *Repository) Metrics(ctx context.Context, now time.Time) (*Metrics, error) {
	m := &Metrics{}
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = $1),
			COUNT(*) FILTER (WHERE status = $2),
			COUNT(*) FILTER (WHERE status = $3),
			COUNT(*) FILTER (WHERE status = $4),
			COUNT(*) FILTER (WHERE status = $5),
			COALESCE(SUM(attempts), 0)
		FROM background_jobs
	`, StatusQueued, StatusProcessing, StatusRetryable, StatusFailed, StatusCompleted).Scan(
		&m.Queued, &m.Processing, &m.Retryable, &m.Failed, &m.Completed, &m.TotalRetries,
	)
	if err != nil {
		return nil, fmt.Errorf("job metrics: %w", err)
	}

	var oldest *time.Time
	_ = r.pool.QueryRow(ctx, `
		SELECT MIN(created_at) FROM background_jobs WHERE status IN ($1, $2)
	`, StatusQueued, StatusRetryable).Scan(&oldest)
	if oldest != nil {
		m.OldestQueuedAge = now.Sub(*oldest)
	}
	return m, nil
}

// IsUniqueViolation checks for PostgreSQL unique constraint errors.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
