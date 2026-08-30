package researchverification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// QueueItem is a prioritized NSF REU candidate awaiting or due for verification.
type QueueItem struct {
	ID               uuid.UUID       `json:"id"`
	Title            string          `json:"title"`
	OrganizationName string          `json:"organization_name"`
	SourceURL        *string         `json:"source_url"`
	ProgramURL       *string         `json:"program_url"`
	ApplicationStatus string         `json:"application_status"`
	TypeMetadata     json.RawMessage `json:"type_metadata"`
	PriorityScore    int             `json:"priority_score"`
	LastVerifiedAt   *time.Time      `json:"last_verified_at"`
	NextVerificationAt *time.Time    `json:"next_verification_at"`
}

// VerificationRecord is a persisted availability verification event.
type VerificationRecord struct {
	ID                    uuid.UUID  `json:"id"`
	OpportunityID         uuid.UUID  `json:"opportunity_id"`
	ApplicationStatus     string     `json:"application_status"`
	ApplicationURL        *string    `json:"application_url"`
	VerificationSourceURL *string    `json:"verification_source_url"`
	OpensAt               *time.Time `json:"opens_at"`
	Deadline              *time.Time `json:"deadline"`
	CycleLabel            *string    `json:"cycle_label"`
	VerificationMethod    string     `json:"verification_method"`
	VerifiedAt            time.Time  `json:"verified_at"`
	VerifiedBy            *uuid.UUID `json:"verified_by"`
	NextVerificationAt    *time.Time `json:"next_verification_at"`
	Notes                 *string    `json:"notes"`
}

// VerifyRequest is the admin submission payload.
type VerifyRequest struct {
	ApplicationStatus     string  `json:"application_status"`
	ApplicationURL        *string `json:"application_url"`
	VerificationSourceURL string  `json:"verification_source_url"`
	OpensAt               *string `json:"opens_at"`
	Deadline              *string `json:"deadline"`
	CycleLabel            *string `json:"cycle_label"`
	VerificationMethod    string  `json:"verification_method"`
	Notes                 *string `json:"notes"`
}

// Metrics summarizes research verification catalog state.
type Metrics struct {
	CandidatePrograms          int `json:"candidate_nsf_reu_programs"`
	AvailabilityUnknown        int `json:"availability_unknown"`
	VerifiedPrograms           int `json:"verified_programs"`
	ApplicationsOpen           int `json:"applications_open"`
	ApplicationsUpcoming       int `json:"applications_upcoming"`
	ApplicationsClosed         int `json:"applications_closed"`
	DirectApplicationURLs      int `json:"direct_application_urls_verified"`
	VerifiedDeadlines          int `json:"verified_deadlines"`
	VerificationStale          int `json:"verification_stale"`
}
