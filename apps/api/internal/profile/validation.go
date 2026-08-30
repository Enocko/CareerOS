package profile

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/careeros/api/internal/platform"
)

const (
	maxNameLength       = 100
	minGraduationYear   = 2020
	maxGraduationYear   = 2040
	defaultUniversity   = "Grambling State University"
)

var (
	validWorkArrangements = map[string]bool{
		"remote": true, "hybrid": true, "on_site": true, "flexible": true,
	}
	validExperienceLevels = map[string]bool{
		"intern": true, "entry": true, "mid": true, "senior": true,
	}
)

// ValidateUpdateRequest validates profile update input.
func ValidateUpdateRequest(req UpdateRequest) error {
	var details []platform.FieldError

	if req.FirstName != nil && len(*req.FirstName) > maxNameLength {
		details = append(details, platform.FieldError{Field: "first_name", Message: fmt.Sprintf("must be at most %d characters", maxNameLength)})
	}
	if req.LastName != nil && len(*req.LastName) > maxNameLength {
		details = append(details, platform.FieldError{Field: "last_name", Message: fmt.Sprintf("must be at most %d characters", maxNameLength)})
	}

	if req.GraduationYear != nil {
		year := *req.GraduationYear
		if year < minGraduationYear || year > maxGraduationYear {
			details = append(details, platform.FieldError{
				Field:   "graduation_year",
				Message: fmt.Sprintf("must be between %d and %d", minGraduationYear, maxGraduationYear),
			})
		}
	}

	if req.WorkArrangement != nil && *req.WorkArrangement != "" {
		if !validWorkArrangements[*req.WorkArrangement] {
			details = append(details, platform.FieldError{
				Field:   "work_arrangement",
				Message: "must be one of: remote, hybrid, on_site, flexible",
			})
		}
	}

	if req.ExperienceLevel != nil && *req.ExperienceLevel != "" {
		if !validExperienceLevels[*req.ExperienceLevel] {
			details = append(details, platform.FieldError{
				Field:   "experience_level",
				Message: "must be one of: intern, entry, mid, senior",
			})
		}
	}

	for _, field := range []struct {
		name string
		val  *string
	}{
		{"github_url", req.GithubURL},
		{"linkedin_url", req.LinkedinURL},
		{"portfolio_url", req.PortfolioURL},
	} {
		if field.val != nil && *field.val != "" {
			if err := validateURL(*field.val); err != nil {
				details = append(details, platform.FieldError{Field: field.name, Message: "must be a valid URL"})
			}
		}
	}

	if len(details) > 0 {
		return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Validation failed").WithDetails(details)
	}

	return nil
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https")
	}
	return nil
}

func normalizeStringSlice(slice []string) []string {
	if slice == nil {
		return []string{}
	}
	return slice
}

func stringOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func universityOrDefault(s *string) string {
	if s == nil {
		return defaultUniversity
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return defaultUniversity
	}
	return trimmed
}
