package jobs_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/careeros/api/internal/jobs"
	"github.com/google/uuid"
)

func TestQueueStressConcurrentClaims(t *testing.T) {
	repo, pool := testRepo(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM background_jobs`)

	const total = 40
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		key := "test:stress:" + uuid.NewString()
		id, created, err := repo.Enqueue(ctx, jobs.TypeDeadlineReminder, map[string]int{"i": i}, key, time.Now().UTC(), 3)
		if err != nil || !created {
			t.Fatalf("enqueue %d: created=%v err=%v", i, created, err)
		}
		ids = append(ids, id.String())
	}

	var claimed sync.Map
	var duplicates atomic.Int32
	var completed atomic.Int32

	var wg sync.WaitGroup
	workers := 4
	deadline := time.After(5 * time.Second)
	done := make(chan struct{})
	go func() {
		<-deadline
		close(done)
	}()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					now := time.Now().UTC()
					job, err := repo.ClaimNext(ctx, workerID, now)
					if err != nil {
						t.Error(err)
						return
					}
					if job == nil {
						time.Sleep(5 * time.Millisecond)
						continue
					}
					if _, loaded := claimed.LoadOrStore(job.ID.String(), true); loaded {
						duplicates.Add(1)
					}
					if err := repo.MarkCompleted(ctx, job.ID, now); err != nil {
						t.Error(err)
						return
					}
					completed.Add(1)
				}
			}
		}("stress-" + uuid.NewString()[:4])
	}
	wg.Wait()

	if duplicates.Load() > 0 {
		t.Fatalf("duplicate claims observed: %d", duplicates.Load())
	}
	if int(completed.Load()) < total {
		t.Fatalf("expected at least %d completed, got %d", total, completed.Load())
	}

	var stressCompleted int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM background_jobs
		WHERE id = ANY($1::uuid[]) AND status = $2
	`, ids, jobs.StatusCompleted).Scan(&stressCompleted)
	if err != nil {
		t.Fatal(err)
	}
	if stressCompleted != total {
		t.Fatalf("expected %d stress jobs completed, got %d", total, stressCompleted)
	}
}
