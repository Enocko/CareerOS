package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/careeros/api/internal/jobs"
	"github.com/google/uuid"
)

func TestReclaimStaleProcessing(t *testing.T) {
	repo, pool := testRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	key := "test:stale:" + uuid.NewString()

	id, created, err := repo.Enqueue(ctx, jobs.TypeDeadlineReminder, map[string]string{}, key, now, 3)
	if err != nil || !created {
		t.Fatalf("enqueue: %v created=%v", err, created)
	}

	job, err := repo.ClaimNext(ctx, "worker-a", now)
	if err != nil || job == nil {
		t.Fatalf("claim: %v", err)
	}

	staleLockedAt := now.Add(-20 * time.Minute)
	_, err = pool.Exec(ctx, `
		UPDATE background_jobs SET locked_at = $2 WHERE id = $1
	`, id, staleLockedAt)
	if err != nil {
		t.Fatal(err)
	}

	reclaimed, err := repo.ReclaimStaleProcessing(ctx, now, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed != 1 {
		t.Fatalf("expected 1 reclaimed job, got %d", reclaimed)
	}

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM background_jobs WHERE id = $1`, id).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}
	if status != jobs.StatusRetryable {
		t.Fatalf("expected retryable, got %s", status)
	}

	job2, err := repo.ClaimNext(ctx, "worker-b", now)
	if err != nil || job2 == nil || job2.ID != id {
		t.Fatalf("expected reclaimed job to be claimable, got %v err=%v", job2, err)
	}
}
