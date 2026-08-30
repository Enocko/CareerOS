package recommendation

import (
	"time"

	"github.com/careeros/api/internal/opportunities"
	"github.com/google/uuid"
)

// Factor describes one scored component for explainability.
type Factor struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Points float64 `json:"points"`
	Max    float64 `json:"max"`
}

// Result is a ranked opportunity with score and explanations.
type Result struct {
	Opportunity opportunities.Summary `json:"opportunity"`
	MatchScore  int                   `json:"match_score"`
	Factors     []Factor              `json:"factors"`
	Reasons     []string              `json:"match_reasons"`
	ReasonShort string                `json:"match_summary,omitempty"`
}

// Candidate is an opportunity eligible for scoring.
type Candidate struct {
	ID                 uuid.UUID
	Title              string
	OrganizationName   string
	Category           string
	Location           *string
	WorkArrangement    string
	Deadline           *time.Time
	Skills             []string
	Tags               []string
	Status             string
	VerificationStatus string
	SourceName         string
	LastCheckedAt      *time.Time
	ExperienceLevel    *string
	CareerFamily       *string
	EducationLevel     *string
	RelevanceTier      *string
	IsSaved            bool
	HasApplication     bool
}

// StudentContext is the normalized profile input for scoring.
type StudentContext struct {
	Major              string
	GraduationYear     *int
	CareerInterests    []string
	DesiredRoles       []string
	Skills             []string
	Technologies       []string
	PreferredLocations []string
	WorkArrangement    string
	ExperienceLevel    string
	InferredFamilies   []string
	PreferredExpLevels []string
	HasProfile         bool
	ProfileComplete    bool
}

// ListFilter holds pagination for recommendations.
type ListFilter struct {
	Page    int
	PerPage int
}
