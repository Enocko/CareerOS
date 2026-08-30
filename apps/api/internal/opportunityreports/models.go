package opportunityreports

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending   = "pending"
	StatusResolved  = "resolved"
	StatusDismissed = "dismissed"

	ReasonAppearsClosed      = "appears_closed"
	ReasonBrokenLink         = "broken_link"
	ReasonIncorrectDeadline  = "incorrect_deadline"
	ReasonDuplicate          = "duplicate"
	ReasonIncorrectInfo      = "incorrect_info"
	ReasonOther              = "other"
)

var ValidReasons = map[string]bool{
	ReasonAppearsClosed:     true,
	ReasonBrokenLink:        true,
	ReasonIncorrectDeadline: true,
	ReasonDuplicate:         true,
	ReasonIncorrectInfo:     true,
	ReasonOther:             true,
}

var ValidStatuses = map[string]bool{
	StatusPending:   true,
	StatusResolved:  true,
	StatusDismissed: true,
}

// Report is a stored student issue report.
type Report struct {
	ID            uuid.UUID  `json:"id"`
	OpportunityID uuid.UUID  `json:"opportunity_id"`
	ReporterID    uuid.UUID  `json:"reporter_id"`
	Reason        string     `json:"reason"`
	Note          *string    `json:"note,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy    *uuid.UUID `json:"resolved_by,omitempty"`
}

// CreateRequest is the student report payload.
type CreateRequest struct {
	Reason string  `json:"reason"`
	Note   *string `json:"note,omitempty"`
}

// UpdateStatusRequest is the admin triage payload.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// QueueItem is an admin view of grouped pending reports.
type QueueItem struct {
	OpportunityID     uuid.UUID  `json:"opportunity_id"`
	OpportunityTitle  string     `json:"opportunity_title"`
	OrganizationName  string     `json:"organization_name"`
	SourceName        string     `json:"source_name"`
	SourceURL         *string    `json:"source_url,omitempty"`
	OpportunityStatus string     `json:"opportunity_status"`
	VerificationStatus string    `json:"verification_status"`
	LastCheckedAt     *time.Time `json:"last_checked_at,omitempty"`
	ReportCount       int        `json:"report_count"`
	LatestReportAt    time.Time  `json:"latest_report_at"`
	Reasons           []string   `json:"reasons"`
}
