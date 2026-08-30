package notifications

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Service handles notification business logic.
type Service struct {
	repo *Repository
}

// NewService creates a notification service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateDeadlineReminder creates an in-app deadline reminder notification.
func (s *Service) CreateDeadlineReminder(ctx context.Context, input DeadlineReminderInput) (bool, error) {
	notifType := TypeApplicationDeadline
	title := formatReminderTitle(input.ReminderKind, input.WindowDays)
	message := formatReminderMessage(input)

	return s.repo.CreateIdempotent(ctx, Notification{
		UserID:        input.UserID,
		Type:          notifType,
		Title:         title,
		Message:       message,
		OpportunityID: &input.OpportunityID,
		ApplicationID: input.ApplicationID,
	}, input.IdempotencyKey)
}

func formatReminderTitle(kind string, window int) string {
	if kind == "saved" {
		return formatDays("Saved opportunity deadline", window)
	}
	return formatDays("Application deadline", window)
}

func formatDays(prefix string, window int) string {
	switch window {
	case 1:
		return prefix + " tomorrow"
	default:
		return prefix + " in " + strconv.Itoa(window) + " days"
	}
}

func formatReminderMessage(input DeadlineReminderInput) string {
	if input.ReminderKind == "saved" {
		return "You saved \"" + input.OpportunityTitle + "\" — the deadline is " + input.DeadlineDate + " and you have not started tracking an application yet."
	}
	return "\"" + input.OpportunityTitle + "\" is due on " + input.DeadlineDate + "."
}

// List returns notifications for a user.
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit, offset)
}

// UnreadCount returns unread notifications for a user.
func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead marks a notification read.
func (s *Service) MarkRead(ctx context.Context, userID, notificationID uuid.UUID, now time.Time) error {
	return s.repo.MarkRead(ctx, userID, notificationID, now)
}

// MarkAllRead marks all notifications read.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID, now time.Time) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID, now)
}
