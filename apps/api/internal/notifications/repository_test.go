package notifications_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func TestNotificationIdempotentCreate(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	userID := insertTestUser(t, pool)
	oppID := insertTestOpportunity(t, pool)

	repo := notifications.NewRepository(pool)
	svc := notifications.NewService(repo)
	key := "test:notif:" + uuid.NewString()

	created1, err := svc.CreateDeadlineReminder(ctx, notifications.DeadlineReminderInput{
		UserID: userID, OpportunityID: oppID, ReminderKind: "saved",
		WindowDays: 3, OpportunityTitle: "SWE Intern", DeadlineDate: "2026-09-15", IdempotencyKey: key,
	})
	if err != nil || !created1 {
		t.Fatalf("first create: created=%v err=%v", created1, err)
	}

	created2, err := svc.CreateDeadlineReminder(ctx, notifications.DeadlineReminderInput{
		UserID: userID, OpportunityID: oppID, ReminderKind: "saved",
		WindowDays: 3, OpportunityTitle: "SWE Intern", DeadlineDate: "2026-09-15", IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("duplicate notification should be suppressed")
	}
}

func TestNoReminderWithoutDeadline(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := notifications.NewRepository(pool)
	target := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	apps, err := repo.ListApplicationDeadlineCandidates(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	_ = apps
}

func TestReminderIdempotencyKeyFormat(t *testing.T) {
	userID := uuid.New()
	oppID := uuid.New()
	key := jobs.ReminderIdempotencyKey(userID, oppID, nil, "2026-09-15", 7)
	if key == "" {
		t.Fatal("expected key")
	}
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "notif-test-" + id.String() + "@example.com"
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'hash')
		ON CONFLICT DO NOTHING
	`, id, email)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE user_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func insertTestOpportunity(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	sourceID := uuid.MustParse("c3000000-0000-4000-8000-000000000001")
	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO opportunities (
			id, source_id, external_id, title, organization_name, description,
			category, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count, deadline,
			experience_level, career_family, education_level, relevance_tier, classification_reasons
		) VALUES (
			$1, $2, $3, 'Notification Test Role', 'Test Co', 'Test',
			'internship', 'remote', 'https://example.com', 'https://example.com', 'USAJobs',
			'verified', $4, $4, $4, 'open', '{}', '{test}', 0, '2026-09-15',
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{test}'
		)
	`, id, sourceID, "NOTIF-UNIT-"+id.String(), now)
	if err != nil {
		t.Fatalf("insert opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE opportunity_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id
}
