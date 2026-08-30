package opportunitytype

import (
	"encoding/json"
	"testing"
)

func TestValidateWrite_ResearchApplicationStatus(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Research,
		TypeMetadata: json.RawMessage(`{
			"application_status": "unknown",
			"application_status_method": "nsf_award_only",
			"availability_verification_method": "unknown",
			"program_url": "https://example.edu/reu"
		}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrite_ResearchInvalidApplicationStatus(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Research,
		TypeMetadata:    json.RawMessage(`{"application_status":"maybe"}`),
	})
	if err == nil {
		t.Fatal("expected invalid application_status error")
	}
}

func TestDefaultResearchAvailabilityMetadata(t *testing.T) {
	meta := DefaultResearchAvailabilityMetadata()
	if meta["application_status"] != ApplicationStatusUnknown {
		t.Fatalf("expected unknown, got %q", meta["application_status"])
	}
}
