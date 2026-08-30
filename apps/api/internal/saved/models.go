package saved

import (
	"time"

	"github.com/careeros/api/internal/opportunities"
	"github.com/google/uuid"
)

// SavedOpportunity is the response for a saved opportunity bookmark.
type SavedOpportunity struct {
	ID            uuid.UUID `json:"id"`
	OpportunityID uuid.UUID `json:"opportunity_id"`
	SavedAt       time.Time `json:"saved_at"`
}

// ListFilter holds query parameters for listing saved opportunities.
type ListFilter struct {
	Page    int
	PerPage int
}

// Summary wraps an opportunity summary from a saved list.
type Summary = opportunities.Summary
