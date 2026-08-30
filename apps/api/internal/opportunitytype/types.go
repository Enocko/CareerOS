package opportunitytype

// Canonical opportunity types (Model C).
const (
	Employment   = "employment"
	Research     = "research"
	Scholarship  = "scholarship"
	Fellowship   = "fellowship"
	Program      = "program"
	Event        = "event"
	Competition  = "competition"
	Other        = "other"
)

// Verification methods complement verification_status.
const (
	VerificationOfficialSource   = "official_source"
	VerificationPartner          = "partner"
	VerificationManualVerified   = "manual_verified"
	VerificationCommunityVerified = "community_verified"
)

// Employment modes (employment opportunities only).
const (
	EmploymentModeFullTime = "full_time"
	EmploymentModePartTime = "part_time"
	EmploymentModeSeasonal = "seasonal"
)

var validTypes = map[string]struct{}{
	Employment:  {},
	Research:    {},
	Scholarship: {},
	Fellowship:  {},
	Program:     {},
	Event:       {},
	Competition: {},
	Other:       {},
}

var validVerificationMethods = map[string]struct{}{
	VerificationOfficialSource:    {},
	VerificationPartner:           {},
	VerificationManualVerified:    {},
	VerificationCommunityVerified: {},
}

var validEmploymentModes = map[string]struct{}{
	EmploymentModeFullTime: {},
	EmploymentModePartTime: {},
	EmploymentModeSeasonal: {},
}

var validExperienceLevels = map[string]struct{}{
	"internship":     {},
	"co_op":          {},
	"new_grad":       {},
	"early_career":   {},
	"apprenticeship": {},
	"fellowship":     {},
	"unknown":        {},
}

// ValidType reports whether t is a supported opportunity_type.
func ValidType(t string) bool {
	_, ok := validTypes[t]
	return ok
}

// FromLegacyCategory maps deprecated category values to Model C opportunity_type.
func FromLegacyCategory(category string) string {
	switch category {
	case Fellowship:
		return Fellowship
	case Scholarship:
		return Scholarship
	case Research:
		return Research
	case "hackathon", "conference":
		return Event
	case "leadership_program":
		return Program
	case "internship", "full_time", "part_time", "apprenticeship":
		return Employment
	default:
		return Employment
	}
}

// EmploymentModeFromLegacyCategory derives employment_mode from legacy category.
func EmploymentModeFromLegacyCategory(category string) string {
	switch category {
	case "full_time":
		return EmploymentModeFullTime
	case "part_time":
		return EmploymentModePartTime
	default:
		return ""
	}
}

// IngestionDefaults returns universal fields for employment ingestion adapters.
func IngestionDefaults(category string) (opportunityType, employmentMode, verificationMethod string) {
	return Employment, EmploymentModeFromLegacyCategory(category), VerificationOfficialSource
}

// IsNonEmploymentIngestion reports whether raw ingestion should bypass employment classification.
func IsNonEmploymentIngestion(opportunityType string) bool {
	return opportunityType != "" && opportunityType != Employment
}
