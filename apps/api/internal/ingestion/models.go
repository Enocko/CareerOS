package ingestion

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	VerificationVerified   = "verified"
	VerificationUnverified = "unverified"
	VerificationStale      = "stale"
	VerificationClosed     = "closed"

	RunStatusRunning = "running"
	RunStatusSuccess = "success"
	RunStatusFailed  = "failed"

	StaleAfterMissedSyncs = 2
)

// Source represents a registered opportunity ingestion source.
type Source struct {
	ID                  uuid.UUID       `json:"id"`
	Name                string          `json:"name"`
	SourceType          string          `json:"source_type"`
	Adapter             string          `json:"adapter"`
	Config              json.RawMessage `json:"config"`
	Enabled             bool            `json:"enabled"`
	SyncIntervalMinutes int             `json:"sync_interval_minutes"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// Run records a single ingestion execution for a source.
type Run struct {
	ID                 uuid.UUID  `json:"id"`
	SourceID           uuid.UUID  `json:"source_id"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at"`
	Status             string     `json:"status"`
	RecordsRawFetched  int        `json:"records_raw_fetched"`
	RecordsRetained    int        `json:"records_retained"`
	RecordsFilteredOut int        `json:"records_filtered_out"`
	RecordsFetched     int        `json:"records_fetched"`
	RecordsCreated     int        `json:"records_created"`
	RecordsUpdated     int        `json:"records_updated"`
	RecordsStale       int        `json:"records_stale"`
	RecordsClosed      int        `json:"records_closed"`
	ErrorMessage       *string    `json:"error_message"`
	ErrorCode          *string    `json:"error_code"`
}

// RunResult summarizes the outcome of a completed ingestion run.
type RunResult struct {
	RunID              uuid.UUID
	SourceName         string
	Status             string
	RecordsRawFetched  int
	RecordsRetained    int
	RecordsFilteredOut int
	RecordsFetched     int // deprecated alias for RecordsRetained
	RecordsCreated     int
	RecordsUpdated     int
	RecordsStale       int
	RecordsClosed      int
	ErrorMessage       string
}
