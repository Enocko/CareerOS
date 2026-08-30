package relevance

import (
	"regexp"
	"strings"
)

// Filter reason constants for ingestion diagnostics.
const (
	ReasonInternshipTitleMatch     = "internship_title_match"
	ReasonNewGradMatch             = "new_grad_match"
	ReasonEarlyCareerMatch         = "early_career_match"
	ReasonStudentProgramMatch      = "student_program_match"
	ReasonSeniorTitleExclusion     = "senior_title_exclusion"
	ReasonExperienceExclusion      = "experience_requirement_exclusion"
	ReasonRecruitingRoleExclusion  = "recruiting_role_exclusion"
	ReasonNonRelevantRole          = "non_relevant_role"
	ReasonNotListedExclusion       = "not_listed_exclusion"
	ReasonEmptyTitle               = "empty_title"
	ReasonUnknown                  = "unknown"
)

var (
	earlyCareerPattern = regexp.MustCompile(`(?i)\b(intern(ship)?|co-?op|new\s?grad(uate)?|early\s+career|campus)\b`)
	techRolePattern    = regexp.MustCompile(`(?i)\b(software|engineer(ing)?|developer|data\s*science|machine\s*learning|\bml\b|cyber\s*security|security|product|sre|devops|cloud|backend|frontend|full[\s-]?stack)\b`)
	excludePattern     = regexp.MustCompile(`(?i)\b(recruiter|recruiting|talent\s+acquisition|university\s+relations|campus\s+recruiter|hiring\s+manager)\b`)
	internPattern      = regexp.MustCompile(`(?i)\bintern(ship)?\b`)
	coopPattern        = regexp.MustCompile(`(?i)\bco-?op\b`)
	newGradPattern     = regexp.MustCompile(`(?i)\bnew\s?grad(uate)?\b`)
	campusPattern      = regexp.MustCompile(`(?i)\b(campus|early\s+career)\b`)
)

// Decision captures relevance classification and the diagnostic reason.
type Decision struct {
	Relevant bool
	Reason   string
}

// ClassifyStudentRelevance applies conservative rule-based filtering for internship and early-career roles.
func ClassifyStudentRelevance(title string) Decision {
	title = strings.TrimSpace(title)
	if title == "" {
		return Decision{Relevant: false, Reason: ReasonEmptyTitle}
	}

	if excludePattern.MatchString(title) && !earlyCareerPattern.MatchString(title) {
		return Decision{Relevant: false, Reason: ReasonRecruitingRoleExclusion}
	}

	if earlyCareerPattern.MatchString(title) {
		switch {
		case internPattern.MatchString(title):
			return Decision{Relevant: true, Reason: ReasonInternshipTitleMatch}
		case coopPattern.MatchString(title):
			return Decision{Relevant: true, Reason: ReasonEarlyCareerMatch}
		case newGradPattern.MatchString(title):
			return Decision{Relevant: true, Reason: ReasonNewGradMatch}
		case campusPattern.MatchString(title):
			return Decision{Relevant: true, Reason: ReasonStudentProgramMatch}
		default:
			return Decision{Relevant: true, Reason: ReasonEarlyCareerMatch}
		}
	}

	lower := strings.ToLower(title)
	if techRolePattern.MatchString(title) {
		if strings.Contains(lower, "new grad") || strings.Contains(lower, "university") {
			return Decision{Relevant: true, Reason: ReasonNewGradMatch}
		}
	}

	return Decision{Relevant: false, Reason: ReasonNonRelevantRole}
}

// IsStudentRelevant reports whether a title matches student-relevant rules.
func IsStudentRelevant(title string) bool {
	return ClassifyStudentRelevance(title).Relevant
}
