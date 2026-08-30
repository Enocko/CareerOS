package opportunities

import (
	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/opportunitytype"
)

// isBrowseDiscoverable mirrors default Browse filters (employment technical-only, no unverified).
func isBrowseDiscoverable(o *Opportunity, externalID string) bool {
	if o.Status != "open" {
		return false
	}
	if o.VerificationStatus != ingestion.VerificationVerified &&
		o.VerificationStatus != ingestion.VerificationUnverified {
		return false
	}
	if externalID != "" && ingestion.IsTestExternalID(externalID) {
		return false
	}

	switch o.OpportunityType {
	case opportunitytype.Research:
		return true
	case opportunitytype.Employment:
		return o.RelevanceTier != nil && *o.RelevanceTier == "high_confidence_technical"
	default:
		return false
	}
}

// canAccessDetail reports whether a student may view a non-discoverable opportunity.
func canAccessDetail(o *Opportunity, externalID string, hasViewed bool) bool {
	if isBrowseDiscoverable(o, externalID) {
		return true
	}
	return o.IsSaved || o.HasApplication || hasViewed
}
