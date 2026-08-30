package jobs_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReminderSchedulerTickEmptyCatalog(t *testing.T) {
	jobsRepo, _ := testRepo(t)
	ctx := context.Background()
	pool := testPool(t)
	notifRepo := notifications.NewRepository(pool)
	scheduler := jobs.NewScheduler(jobsRepo, nil, notifRepo, jobs.DefaultConfig())
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	if _, err := scheduler.EnqueueDeadlineReminders(ctx, now); err != nil {
		t.Fatalf("enqueue reminders: %v", err)
	}
}

func TestReminderWindowsEnqueue7DayApplication(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	deadline := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, 7)

	userID := insertReminderUser(t, pool)
	oppID := insertReminderOpportunity(t, pool, deadline, "open", true)
	appID := insertReminderApplication(t, pool, userID, oppID)

	jobsRepo := jobs.NewRepository(pool)
	ingestRepo := ingestion.NewRepository(pool)
	notifRepo := notifications.NewRepository(pool)
	scheduler := jobs.NewScheduler(jobsRepo, ingestRepo, notifRepo, jobs.DefaultConfig())

	count, err := scheduler.EnqueueDeadlineReminders(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("expected at least one reminder job, got %d", count)
	}

	// Duplicate scheduler tick should not enqueue another active job for same window.
	count2, err := scheduler.EnqueueDeadlineReminders(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if count2 != 0 {
		t.Fatalf("expected duplicate enqueue suppression, got %d new jobs", count2)
	}

	job, err := claimReminderJobForApp(ctx, jobsRepo, appID, now)
	if err != nil {
		t.Fatal(err)
	}

	var payload jobs.DeadlineReminderPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ApplicationID == nil || *payload.ApplicationID != appID {
		t.Fatalf("expected application reminder for %s", appID)
	}
	if payload.WindowDays != 7 {
		t.Fatalf("expected 7-day window, got %d", payload.WindowDays)
	}
}

func TestNoReminderForClosedOpportunity(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	deadline := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, 3)

	userID := insertReminderUser(t, pool)
	oppID := insertReminderOpportunity(t, pool, deadline, "closed", true)
	insertReminderApplication(t, pool, userID, oppID)

	notifRepo := notifications.NewRepository(pool)
	rows, err := notifRepo.ListApplicationDeadlineCandidates(ctx, deadline)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.OpportunityID == oppID {
			t.Fatal("closed opportunity should not be a reminder candidate")
		}
	}
}

func TestNoReminderWithoutDeadline(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	target := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	userID := insertReminderUser(t, pool)
	oppID := insertReminderOpportunity(t, pool, time.Time{}, "open", false)
	insertReminderApplication(t, pool, userID, oppID)

	notifRepo := notifications.NewRepository(pool)
	rows, err := notifRepo.ListApplicationDeadlineCandidates(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.OpportunityID == oppID {
			t.Fatal("opportunity without deadline should not be a candidate")
		}
	}
}

func TestSavedReminderExcludedWhenApplicationExists(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	deadline := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, 1)

	userID := insertReminderUser(t, pool)
	oppID := insertReminderOpportunity(t, pool, deadline, "open", true)
	insertSavedOpportunity(t, pool, userID, oppID)
	insertReminderApplication(t, pool, userID, oppID)

	notifRepo := notifications.NewRepository(pool)
	rows, err := notifRepo.ListSavedDeadlineCandidates(ctx, deadline)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.OpportunityID == oppID {
			t.Fatal("saved reminder should be suppressed when application exists")
		}
	}
}

func TestIngestOverlapProtection(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	jobsRepo := jobs.NewRepository(pool)
	ingestRepo := ingestion.NewRepository(pool)
	sourceID := uuid.MustParse("c3000000-0000-4000-8000-000000000001")

	_, _ = pool.Exec(ctx, `DELETE FROM background_jobs WHERE idempotency_key = $1`, jobs.IngestIdempotencyKey(sourceID))
	_, err := pool.Exec(ctx, `
		INSERT INTO ingestion_runs (id, source_id, status, started_at)
		VALUES ($1, $2, $3, now())
	`, uuid.New(), sourceID, ingestion.RunStatusRunning)
	if err != nil {
		t.Fatalf("insert running ingestion: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM ingestion_runs WHERE source_id = $1 AND finished_at IS NULL`, sourceID)
	})

	running, err := ingestRepo.HasRunningIngestion(ctx, sourceID)
	if err != nil || !running {
		t.Fatalf("expected running ingestion guard, running=%v err=%v", running, err)
	}

	id, created, err := jobsRepo.Enqueue(ctx, jobs.TypeIngestSource, jobs.IngestSourcePayload{SourceID: sourceID},
		jobs.IngestIdempotencyKey(sourceID), time.Now().UTC(), 5)
	if err != nil || !created {
		t.Fatalf("enqueue ingest job: id=%v created=%v err=%v", id, created, err)
	}
	active, err := jobsRepo.HasActiveIngestJob(ctx, sourceID)
	if err != nil || !active {
		t.Fatalf("expected active ingest job, active=%v err=%v", active, err)
	}

	_, created2, err := jobsRepo.Enqueue(ctx, jobs.TypeIngestSource, jobs.IngestSourcePayload{SourceID: sourceID},
		jobs.IngestIdempotencyKey(sourceID), time.Now().UTC(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected duplicate ingest enqueue to be suppressed")
	}
}

func TestBackoffIncreasesWithAttempts(t *testing.T) {
	cfg := jobs.DefaultConfig()
	b1 := cfg.NextBackoff(1)
	b3 := cfg.NextBackoff(3)
	if b3 <= b1 {
		t.Fatalf("expected backoff to grow: b1=%s b3=%s", b1, b3)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
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
	return pool
}

func insertReminderUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "reminder-" + id.String() + "@example.com"
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'hash')
	`, id, email)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE user_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM applications WHERE student_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM saved_opportunities WHERE student_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func insertReminderOpportunity(t *testing.T, pool *pgxpool.Pool, deadline time.Time, status string, withDeadline bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	sourceID := uuid.MustParse("c3000000-0000-4000-8000-000000000001")
	now := time.Now().UTC()
	var deadlineVal any
	if withDeadline {
		deadlineVal = deadline
	}
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count, deadline,
			experience_level, career_family, education_level, relevance_tier, classification_reasons
		) VALUES (
			$1, $2, $3, 'Reminder Test Role', 'Test Co', 'Test',
			'internship', 'remote', 'https://example.com', 'https://example.com', 'USAJobs',
			'verified', $4, $4, $4, $5, '{}', '{test}', 0, $6,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{test}'
		)
	`, id, sourceID, "REM-"+id.String(), now, status, deadlineVal)
	if err != nil {
		t.Fatalf("insert opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id
}

func insertReminderApplication(t *testing.T, pool *pgxpool.Pool, userID, oppID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO applications (id, student_id, opportunity_id, current_status)
		VALUES ($1, $2, $3, 'applied')
	`, id, userID, oppID)
	if err != nil {
		t.Fatalf("insert application: %v", err)
	}
	return id
}

func insertSavedOpportunity(t *testing.T, pool *pgxpool.Pool, userID, oppID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO saved_opportunities (student_id, opportunity_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, oppID)
	if err != nil {
		t.Fatalf("insert saved opportunity: %v", err)
	}
}

func claimReminderJobForApp(ctx context.Context, repo *jobs.Repository, appID uuid.UUID, now time.Time) (*jobs.Job, error) {
	for i := 0; i < 20; i++ {
		job, err := repo.ClaimNext(ctx, "test-worker", now)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, nil
		}
		var payload jobs.DeadlineReminderPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return nil, err
		}
		if payload.ApplicationID != nil && *payload.ApplicationID == appID {
			return job, nil
		}
		if err := repo.MarkCompleted(ctx, job.ID, now); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
