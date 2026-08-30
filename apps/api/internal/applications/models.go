package applications

import (
	"time"

	"github.com/google/uuid"
)

// ValidStatuses lists all allowed application statuses.
var ValidStatuses = map[string]bool{
	"saved": true, "preparing": true, "applied": true, "oa_assessment": true,
	"interview": true, "final_round": true, "offer": true,
	"rejected": true, "withdrawn": true, "closed": true,
}

// OpportunityBrief is a summary of an opportunity embedded in application responses.
type OpportunityBrief struct {
	ID               uuid.UUID  `json:"id"`
	Title            string     `json:"title"`
	OrganizationName string     `json:"organization_name"`
	Category         string     `json:"category"`
	Deadline         *time.Time `json:"deadline"`
	ApplicationURL   *string    `json:"application_url,omitempty"`
}

// Application is the application tracker record.
type Application struct {
	ID             uuid.UUID         `json:"id"`
	OpportunityID  uuid.UUID         `json:"opportunity_id"`
	CurrentStatus  string            `json:"current_status"`
	DateApplied    *time.Time        `json:"date_applied"`
	Notes          *string           `json:"notes"`
	NextAction     *string           `json:"next_action"`
	NextActionDate *time.Time        `json:"next_action_date"`
	InterviewDate  *time.Time        `json:"interview_date"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Opportunity    *OpportunityBrief `json:"opportunity,omitempty"`
	StatusHistory  []StatusHistory   `json:"status_history,omitempty"`
}

// StatusHistory records a status change.
type StatusHistory struct {
	ID         uuid.UUID `json:"id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ChangedAt  time.Time `json:"changed_at"`
}

// CreateRequest is the payload for POST /api/v1/applications.
type CreateRequest struct {
	OpportunityID uuid.UUID `json:"opportunity_id"`
	Notes         *string   `json:"notes"`
}

// UpdateRequest is the payload for PATCH /api/v1/applications/:id.
type UpdateRequest struct {
	CurrentStatus  *string    `json:"current_status"`
	Notes          *string    `json:"notes"`
	NextAction     *string    `json:"next_action"`
	NextActionDate *time.Time `json:"next_action_date"`
	InterviewDate  *time.Time `json:"interview_date"`
}

// ListFilter holds query parameters for listing applications.
type ListFilter struct {
	Status  string
	Page    int
	PerPage int
}
