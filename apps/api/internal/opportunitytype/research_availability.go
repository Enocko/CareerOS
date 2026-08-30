package opportunitytype

// Application availability statuses (research opportunities only; stored in type_metadata).
// Distinct from verification_status (source/record trust) and opportunities.status (catalog visibility).
const (
	ApplicationStatusOpen     = "open"
	ApplicationStatusUpcoming = "upcoming"
	ApplicationStatusClosed   = "closed"
	ApplicationStatusUnknown  = "unknown"
)

// How application availability was determined.
const (
	AvailabilityMethodUnknown          = "unknown"
	AvailabilityMethodNSFAwardOnly     = "nsf_award_only"
	AvailabilityMethodAutomatedOfficial = "automated_official"
	AvailabilityMethodManualVerified   = "manual_verified"
	AvailabilityMethodManualOfficialPage = "manual_official_page"
	AvailabilityMethodPartnerVerified  = "partner_verified"
)

var validApplicationStatuses = map[string]struct{}{
	ApplicationStatusOpen:     {},
	ApplicationStatusUpcoming: {},
	ApplicationStatusClosed:   {},
	ApplicationStatusUnknown:  {},
}

var validAvailabilityMethods = map[string]struct{}{
	AvailabilityMethodUnknown:           {},
	AvailabilityMethodNSFAwardOnly:      {},
	AvailabilityMethodAutomatedOfficial: {},
	AvailabilityMethodManualVerified:    {},
	AvailabilityMethodManualOfficialPage: {},
	AvailabilityMethodPartnerVerified:   {},
}

// ValidApplicationStatus reports whether s is a supported application_status value.
func ValidApplicationStatus(s string) bool {
	_, ok := validApplicationStatuses[s]
	return ok
}

// ValidAvailabilityMethod reports whether m is a supported availability verification method.
func ValidAvailabilityMethod(m string) bool {
	_, ok := validAvailabilityMethods[m]
	return ok
}

// DefaultResearchAvailabilityMetadata returns type_metadata for NSF award discovery
// before student-facing application availability is verified.
func DefaultResearchAvailabilityMetadata() map[string]string {
	return map[string]string{
		"application_status":               ApplicationStatusUnknown,
		"application_status_method":          AvailabilityMethodNSFAwardOnly,
		"availability_verification_method": AvailabilityMethodUnknown,
	}
}
