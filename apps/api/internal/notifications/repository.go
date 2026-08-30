package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles notification persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a notifications repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Notification is an in-app notification.
type Notification struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	Message       string     `json:"message"`
	OpportunityID *uuid.UUID `json:"opportunity_id"`
	ApplicationID *uuid.UUID `json:"application_id"`
	CreatedAt     time.Time  `json:"created_at"`
	ReadAt        *time.Time `json:"read_at"`
}

// DeadlineReminderInput creates a deadline notification.
type DeadlineReminderInput struct {
	UserID           uuid.UUID
	OpportunityID    uuid.UUID
	ApplicationID    *uuid.UUID
	ReminderKind     string
	WindowDays       int
	OpportunityTitle string
	DeadlineDate     string
	IdempotencyKey   string
}

// CreateIdempotent inserts a notification if the idempotency key is new.
func (r *Repository) CreateIdempotent(ctx context.Context, n Notification, idempotencyKey string) (bool, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notifications (
			user_id, type, title, message, opportunity_id, application_id, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`, n.UserID, n.Type, n.Title, n.Message, n.OpportunityID, n.ApplicationID, idempotencyKey).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("create notification: %w", err)
	}
	return true, nil
}

// List returns paginated notifications for a user.
func (r *Repository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, type, title, message, opportunity_id, application_id, created_at, read_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var items []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.OpportunityID, &n.ApplicationID, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	if items == nil {
		items = []Notification{}
	}
	return items, rows.Err()
}

// CountUnread returns unread notification count.
func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&count)
	return count, err
}

// MarkRead marks one notification read for a user.
func (r *Repository) MarkRead(ctx context.Context, userID, notificationID uuid.UUID, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = $3 WHERE id = $1 AND user_id = $2 AND read_at IS NULL
	`, notificationID, userID, now)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks all notifications read for a user.
func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID, now time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = $2 WHERE user_id = $1 AND read_at IS NULL
	`, userID, now)
	if err != nil {
		return 0, fmt.Errorf("mark all read: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CountCreated returns total notifications created (metrics).
func (r *Repository) CountCreated(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&count)
	return count, err
}

// DeadlineCandidate is eligible for reminder scheduling.
type DeadlineCandidate struct {
	UserID        uuid.UUID
	OpportunityID uuid.UUID
	ApplicationID uuid.UUID
	Title         string
}

// SavedDeadlineCandidate is a saved opportunity reminder candidate.
type SavedDeadlineCandidate struct {
	UserID        uuid.UUID
	OpportunityID uuid.UUID
	Title         string
}

// ListApplicationDeadlineCandidates finds applications with deadline on target date.
func (r *Repository) ListApplicationDeadlineCandidates(ctx context.Context, deadlineDate time.Time) ([]DeadlineCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.student_id, a.id, o.id, o.title
		FROM applications a
		JOIN opportunities o ON o.id = a.opportunity_id
		WHERE o.deadline = $1::date
		  AND o.deadline IS NOT NULL
		  AND o.status = 'open'
		  AND a.current_status NOT IN ('rejected', 'withdrawn', 'closed')
	`, deadlineDate)
	if err != nil {
		return nil, fmt.Errorf("list application deadline candidates: %w", err)
	}
	defer rows.Close()

	var items []DeadlineCandidate
	for rows.Next() {
		var c DeadlineCandidate
		if err := rows.Scan(&c.UserID, &c.ApplicationID, &c.OpportunityID, &c.Title); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []DeadlineCandidate{}
	}
	return items, rows.Err()
}

// ListSavedDeadlineCandidates finds saved opportunities without applications.
func (r *Repository) ListSavedDeadlineCandidates(ctx context.Context, deadlineDate time.Time) ([]SavedDeadlineCandidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT so.student_id, o.id, o.title
		FROM saved_opportunities so
		JOIN opportunities o ON o.id = so.opportunity_id
		WHERE o.deadline = $1::date
		  AND o.deadline IS NOT NULL
		  AND o.status = 'open'
		  AND (
		    o.opportunity_type != 'research'
		    OR COALESCE(o.type_metadata->>'application_status', 'unknown') = 'open'
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM applications a
		      WHERE a.student_id = so.student_id AND a.opportunity_id = o.id
		  )
	`, deadlineDate)
	if err != nil {
		return nil, fmt.Errorf("list saved deadline candidates: %w", err)
	}
	defer rows.Close()

	var items []SavedDeadlineCandidate
	for rows.Next() {
		var c SavedDeadlineCandidate
		if err := rows.Scan(&c.UserID, &c.OpportunityID, &c.Title); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if items == nil {
		items = []SavedDeadlineCandidate{}
	}
	return items, rows.Err()
}
