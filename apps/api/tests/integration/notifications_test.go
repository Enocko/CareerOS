package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNotificationsAPIFlow(t *testing.T) {
	router, pool := setupTestRouterWithPool(t)
	token := registerAndGetToken(t, router)
	userID := getUserIDFromToken(t, router, token)

	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	deadline := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, 3)
	oppID := insertOpportunityWithDeadline(t, pool, deadline, "open")

	notifRepo := notifications.NewRepository(pool)
	notifService := notifications.NewService(notifRepo)
	jobsRepo := jobs.NewRepository(pool)
	processor := jobs.NewProcessor(jobsRepo, nil, nil, notifService, jobs.DefaultConfig())

	appID := insertApplicationForUser(t, pool, userID, oppID)
	deadlineStr := deadline.Format("2006-01-02")
	key := jobs.ReminderIdempotencyKey(userID, oppID, &appID, deadlineStr, 3)

	jobID, created, err := jobsRepo.Enqueue(context.Background(), jobs.TypeDeadlineReminder, jobs.DeadlineReminderPayload{
		UserID: userID, OpportunityID: oppID, ApplicationID: &appID, ReminderKind: "application",
		DeadlineDate: deadlineStr, WindowDays: 3, OpportunityTitle: "Reminder Test Role",
	}, key, now, 5)
	if err != nil || !created {
		t.Fatalf("enqueue reminder job: created=%v err=%v", created, err)
	}

	job, err := jobsRepo.ClaimByID(context.Background(), "integration-test", jobID, now)
	if err != nil || job == nil {
		t.Fatalf("claim job: %v", err)
	}
	if err := processor.Process(context.Background(), job, now); err != nil {
		t.Fatalf("process reminder: %v", err)
	}
	if err := jobsRepo.MarkCompleted(context.Background(), job.ID, now); err != nil {
		t.Fatal(err)
	}

	// Re-enqueue after completion is allowed; notification idempotency prevents duplicates.
	dupJobID, createdDup, err := jobsRepo.Enqueue(context.Background(), jobs.TypeDeadlineReminder, jobs.DeadlineReminderPayload{
		UserID: userID, OpportunityID: oppID, ApplicationID: &appID, ReminderKind: "application",
		DeadlineDate: deadlineStr, WindowDays: 3, OpportunityTitle: "Reminder Test Role",
	}, key, now, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !createdDup {
		t.Fatal("expected completed jobs to allow re-enqueue with same idempotency key")
	}
	dupJob, err := jobsRepo.ClaimByID(context.Background(), "integration-test-2", dupJobID, now)
	if err != nil || dupJob == nil {
		t.Fatalf("claim duplicate job: %v", err)
	}
	if err := processor.Process(context.Background(), dupJob, now); err != nil {
		t.Fatal(err)
	}
	createdAgain, err := notifService.CreateDeadlineReminder(context.Background(), notifications.DeadlineReminderInput{
		UserID: userID, OpportunityID: oppID, ApplicationID: &appID, ReminderKind: "application",
		WindowDays: 3, OpportunityTitle: "Reminder Test Role", DeadlineDate: deadlineStr, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if createdAgain {
		t.Fatal("duplicate notification should be suppressed")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list notifications: %d %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Data   []map[string]any `json:"data"`
		Unread int              `json:"unread"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatal(err)
	}
	if listResp.Unread < 1 {
		t.Fatalf("expected unread notifications, got %d", listResp.Unread)
	}
	if len(listResp.Data) < 1 {
		t.Fatal("expected notification items")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unread count: %d", rec.Code)
	}

	notifID := listResp.Data[0]["id"].(string)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("mark read: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/mark-all-read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark all read: %d", rec.Code)
	}
}

func insertOpportunityWithDeadline(t *testing.T, pool *pgxpool.Pool, deadline time.Time, status string) uuid.UUID {
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
			$1, $2, $3, 'Reminder Test Role', 'Test Co', 'Test',
			'internship', 'remote', 'https://example.com', 'https://example.com', 'USAJobs',
			'verified', $4, $4, $4, $5, '{}', '{test}', 0, $6,
			'internship', 'software_engineering', 'unspecified', 'high_confidence_technical', '{test}'
		)
	`, id, sourceID, "NOTIF-"+id.String(), now, status, deadline)
	if err != nil {
		t.Fatalf("insert opportunity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE opportunity_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM applications WHERE opportunity_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM opportunities WHERE id = $1`, id)
	})
	return id
}

func insertApplicationForUser(t *testing.T, pool *pgxpool.Pool, userID, oppID uuid.UUID) uuid.UUID {
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

