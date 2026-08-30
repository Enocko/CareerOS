package jobs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusRetryable  = "retryable"
	StatusFailed     = "failed"

	TypeIngestSource     = "ingest_source"
	TypeDeadlineReminder = "deadline_reminder"
)

// Job is a durable background work item.
type Job struct {
	ID             uuid.UUID
	JobType        string
	Payload        json.RawMessage
	Status         string
	IdempotencyKey string
	Attempts       int
	MaxAttempts    int
	RunAt          time.Time
	LockedAt       *time.Time
	LockedBy       *string
	LastError      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// IngestSourcePayload targets a single ingestion source.
type IngestSourcePayload struct {
	SourceID uuid.UUID `json:"source_id"`
}

// DeadlineReminderPayload targets a deadline notification.
type DeadlineReminderPayload struct {
	UserID         uuid.UUID  `json:"user_id"`
	OpportunityID  uuid.UUID  `json:"opportunity_id"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty"`
	ReminderKind   string     `json:"reminder_kind"` // application | saved
	DeadlineDate   string     `json:"deadline_date"` // YYYY-MM-DD
	WindowDays     int        `json:"window_days"`
	OpportunityTitle string   `json:"opportunity_title"`
}

// Metrics summarizes queue health.
type Metrics struct {
	Queued          int
	Processing      int
	Retryable       int
	Failed          int
	Completed       int
	OldestQueuedAge time.Duration
	TotalRetries    int
}
