package opportunitytype

import (
	"encoding/json"
	"testing"
)

func TestValidateWrite_EmploymentEmptyMetadata(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType:    Employment,
		ExperienceLevel:    "internship",
		EmploymentMode:     EmploymentModeFullTime,
		VerificationMethod: VerificationOfficialSource,
		TypeMetadata:       json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrite_NonEmploymentRejectsExperienceLevel(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Scholarship,
		ExperienceLevel: "internship",
		TypeMetadata:    json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for experience_level on scholarship")
	}
}

func TestValidateWrite_ResearchValid(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Research,
		TypeMetadata: json.RawMessage(`{
			"research_area": "biology",
			"housing_provided": true,
			"program_start": "2026-06-01",
			"program_end": "2026-08-15"
		}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrite_ResearchInvalidDate(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Research,
		TypeMetadata:    json.RawMessage(`{"program_start":"not-a-date"}`),
	})
	if err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestValidateWrite_ResearchUnsupportedField(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Research,
		TypeMetadata:    json.RawMessage(`{"unknown_field":true}`),
	})
	if err == nil {
		t.Fatal("expected unsupported field error")
	}
}

func TestValidateWrite_ScholarshipValid(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Scholarship,
		TypeMetadata: json.RawMessage(`{
			"award_amount": "up to $5,000",
			"award_amount_max": 5000,
			"renewable": false,
			"fields_of_study": ["computer science"]
		}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrite_ScholarshipInvalidAwardMax(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Scholarship,
		TypeMetadata:    json.RawMessage(`{"award_amount_max": -1}`),
	})
	if err == nil {
		t.Fatal("expected negative award_amount_max error")
	}
}

func TestValidateWrite_ScholarshipInvalidFieldsOfStudy(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Scholarship,
		TypeMetadata:    json.RawMessage(`{"fields_of_study": [""]}`),
	})
	if err == nil {
		t.Fatal("expected empty fields_of_study error")
	}
}

func TestValidateWrite_ProgramEventValid(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Program,
		TypeMetadata: json.RawMessage(`{
			"program_format": "insight",
			"event_start": "2026-10-01",
			"event_end": "2026-10-03",
			"target_class_years": ["sophomore"]
		}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWrite_ProgramEventInvalidDateRange(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Event,
		TypeMetadata: json.RawMessage(`{
			"event_start": "2026-10-10",
			"event_end": "2026-10-01"
		}`),
	})
	if err == nil {
		t.Fatal("expected invalid date range error")
	}
}

func TestValidateWrite_ProgramEventInvalidTargetYears(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Program,
		TypeMetadata:    json.RawMessage(`{"target_class_years":[" "]}`),
	})
	if err == nil {
		t.Fatal("expected invalid target_class_years error")
	}
}

func TestValidateWrite_MalformedJSON(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Employment,
		TypeMetadata:    json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestValidateWrite_UnknownOpportunityType(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: "internship",
		TypeMetadata:    json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected unknown opportunity_type error")
	}
}

func TestFromLegacyCategory(t *testing.T) {
	cases := map[string]string{
		"internship":         Employment,
		"full_time":          Employment,
		"scholarship":        Scholarship,
		"fellowship":         Fellowship,
		"research":           Research,
		"hackathon":          Event,
		"leadership_program": Program,
	}
	for cat, want := range cases {
		if got := FromLegacyCategory(cat); got != want {
			t.Errorf("FromLegacyCategory(%q) = %q, want %q", cat, got, want)
		}
	}
}

func TestValidType_AcceptsFellowshipExperienceLevel(t *testing.T) {
	err := ValidateWrite(WriteInput{
		OpportunityType: Employment,
		ExperienceLevel: "fellowship",
		TypeMetadata:    json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("expected fellowship experience_level to be accepted: %v", err)
	}
}
