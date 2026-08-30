package profile

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a student's career profile.
type Profile struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	FirstName          *string   `json:"first_name"`
	LastName           *string   `json:"last_name"`
	University         *string   `json:"university"`
	Major              *string   `json:"major"`
	GraduationYear     *int      `json:"graduation_year"`
	CareerInterests    []string  `json:"career_interests"`
	DesiredRoles       []string  `json:"desired_roles"`
	Skills             []string  `json:"skills"`
	Technologies       []string  `json:"technologies"`
	PreferredLocations []string  `json:"preferred_locations"`
	WorkArrangement    *string   `json:"work_arrangement"`
	ExperienceLevel    *string   `json:"experience_level"`
	GithubURL          *string   `json:"github_url"`
	LinkedinURL        *string   `json:"linkedin_url"`
	PortfolioURL       *string   `json:"portfolio_url"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// UpdateRequest is the payload for PUT /api/v1/profile.
type UpdateRequest struct {
	FirstName          *string  `json:"first_name"`
	LastName           *string  `json:"last_name"`
	University         *string  `json:"university"`
	Major              *string  `json:"major"`
	GraduationYear     *int     `json:"graduation_year"`
	CareerInterests    []string `json:"career_interests"`
	DesiredRoles       []string `json:"desired_roles"`
	Skills             []string `json:"skills"`
	Technologies       []string `json:"technologies"`
	PreferredLocations []string `json:"preferred_locations"`
	WorkArrangement    *string  `json:"work_arrangement"`
	ExperienceLevel    *string  `json:"experience_level"`
	GithubURL          *string  `json:"github_url"`
	LinkedinURL        *string  `json:"linkedin_url"`
	PortfolioURL       *string  `json:"portfolio_url"`
}
