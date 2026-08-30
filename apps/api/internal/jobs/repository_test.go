package jobs_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/jobs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testRepo(t *testing.T) (*jobs.Repository, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	_, _ = pool.Exec(context.Background(), `DELETE FROM background_jobs WHERE idempotency_key LIKE 'test:%'`)
	return jobs.NewRepository(pool), pool
}

func TestEnqueueIdempotent(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	key := "test:enqueue:" + uuid.NewString()
	id1, created1, err := repo.Enqueue(ctx, jobs.TypeIngestSource, map[string]string{"x": "1"}, key, time.Now().UTC(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if !created1 || id1 == uuid.Nil {
		t.Fatal("expected first enqueue to succeed")
	}
	_, created2, err := repo.Enqueue(ctx, jobs.TypeIngestSource, map[string]string{"x": "2"}, key, time.Now().UTC(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected duplicate enqueue to be suppressed")
	}
}

func TestClaimConcurrentSafety(t *testing.T) {
	repo, pool := testRepo(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM background_jobs WHERE idempotency_key LIKE 'test:%'`)
	key := "test:claim:" + uuid.NewString()
	jobID, created, err := repo.Enqueue(ctx, jobs.TypeDeadlineReminder, map[string]string{"x": "1"}, key, time.Now().UTC(), 3)
	if err != nil || !created {
		t.Fatalf("enqueue: created=%v err=%v", created, err)
	}

	var wg sync.WaitGroup
	claims := make(chan *jobs.Job, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(worker string) {
			defer wg.Done()
			job, err := repo.ClaimByID(ctx, worker, jobID, time.Now().UTC())
			if err != nil {
				t.Error(err)
				return
			}
			if job != nil {
				claims <- job
			}
		}("worker-" + uuid.NewString()[:4])
	}
	wg.Wait()
	close(claims)

	count := 0
	for range claims {
		count++
	}
	if count != 1 {
		t.Fatalf("expected exactly one claim, got %d", count)
	}
	t.Cleanup(func() {
		_ = repo.MarkCompleted(context.Background(), jobID, time.Now().UTC())
	})
}

func TestRetryExhaustionMarksFailed(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	key := "test:retry:" + uuid.NewString()
	id, created, err := repo.Enqueue(ctx, jobs.TypeDeadlineReminder, map[string]string{}, key, time.Now().UTC(), 2)
	if err != nil || !created {
		t.Fatalf("enqueue: created=%v err=%v", created, err)
	}

	job, err := repo.ClaimNext(ctx, "w1", time.Now().UTC())
	if err != nil || job == nil {
		t.Fatalf("claim: %v", err)
	}

	now := time.Now().UTC()
	permanent, err := repo.MarkRetryable(ctx, job.ID, 2, 2, "boom", now.Add(time.Minute), now)
	if err != nil {
		t.Fatal(err)
	}
	if !permanent {
		t.Fatal("expected permanent failure at max attempts")
	}
	_ = id
}

func TestMarkCompleted(t *testing.T) {
	repo, _ := testRepo(t)
	ctx := context.Background()
	key := "test:complete:" + uuid.NewString()
	_, created, err := repo.Enqueue(ctx, jobs.TypeDeadlineReminder, map[string]string{}, key, time.Now().UTC(), 3)
	if err != nil || !created {
		t.Fatal(err)
	}
	job, err := repo.ClaimNext(ctx, "w1", time.Now().UTC())
	if err != nil || job == nil {
		t.Fatal(err)
	}
	if err := repo.MarkCompleted(ctx, job.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}
