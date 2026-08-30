package opportunities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SourceAttribution describes where an opportunity was sourced from.
type SourceAttribution struct {
	Name      string  `json:"name"`
	SourceURL *string `json:"source_url"`
}

// Opportunity is the full opportunity detail.
type Opportunity struct {
	ID                 uuid.UUID          `json:"id"`
	Title              string             `json:"title"`
	OrganizationName   string             `json:"organization_name"`
	Description        string             `json:"description"`
	Category           string             `json:"category"`
	OpportunityType    string             `json:"opportunity_type"`
	TypeMetadata       json.RawMessage    `json:"type_metadata,omitempty"`
	VerificationMethod *string            `json:"verification_method,omitempty"`
	EmploymentMode     *string            `json:"employment_mode,omitempty"`
	Location           *string            `json:"location"`
	WorkArrangement    string             `json:"work_arrangement"`
	Deadline           *time.Time         `json:"deadline"`
	StartDate          *time.Time         `json:"start_date"`
	Eligibility        *string            `json:"eligibility"`
	Skills             []string           `json:"skills"`
	Compensation       *string            `json:"compensation"`
	ApplicationURL     *string            `json:"application_url"`
	Source             SourceAttribution  `json:"source"`
	VerificationStatus string             `json:"verification_status"`
	LastCheckedAt      *time.Time         `json:"last_checked_at"`
	LastSeenAt         *time.Time         `json:"last_seen_at"`
	Status             string             `json:"status"`
	Tags               []string           `json:"tags"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	ExperienceLevel    *string            `json:"experience_level,omitempty"`
	CareerFamily       *string            `json:"career_family,omitempty"`
	EducationLevel     *string            `json:"education_level,omitempty"`
	RelevanceTier      *string            `json:"relevance_tier,omitempty"`
	ClassificationReasons []string       `json:"classification_reasons,omitempty"`
	IsSaved            bool               `json:"is_saved"`
	HasApplication     bool               `json:"has_application"`
}

// Summary is the list-view representation of an opportunity.
type Summary struct {
	ID                 uuid.UUID  `json:"id"`
	Title              string     `json:"title"`
	OrganizationName   string     `json:"organization_name"`
	Category           string          `json:"category"`
	OpportunityType    string          `json:"opportunity_type"`
	VerificationMethod *string         `json:"verification_method,omitempty"`
	EmploymentMode     *string         `json:"employment_mode,omitempty"`
	Location           *string         `json:"location"`
	WorkArrangement    string     `json:"work_arrangement"`
	Deadline           *time.Time `json:"deadline"`
	Skills             []string   `json:"skills"`
	Tags               []string   `json:"tags"`
	Status             string     `json:"status"`
	VerificationStatus string     `json:"verification_status"`
	SourceName         string     `json:"source_name"`
	LastCheckedAt      *time.Time `json:"last_checked_at"`
	IsSaved            bool       `json:"is_saved"`
	HasApplication     bool       `json:"has_application,omitempty"`
	ExperienceLevel    *string    `json:"experience_level,omitempty"`
	CareerFamily       *string    `json:"career_family,omitempty"`
	RelevanceTier      *string         `json:"relevance_tier,omitempty"`
	TypeMetadata       json.RawMessage `json:"type_metadata,omitempty"`
}

// Catalog scope values for unified Browse (query param `type`).
const (
	CatalogScopeAll        = "all"
	CatalogScopeEmployment = "employment"
	CatalogScopeResearch   = "research"
)

// ListFilter holds query parameters for listing opportunities.
type ListFilter struct {
	Query               string
	Category            string
	OpportunityType     string
	CatalogScope        string
	ApplicationStatus   string
	WorkArrangement     string
	Location            string
	CareerFamily        string
	ExperienceLevel     string
	IncludeUnverified   bool
	IncludeAmbiguous    bool
	IncludeNonTechnical bool
	Page                int
	PerPage             int
}
