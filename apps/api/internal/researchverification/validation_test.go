package researchverification

import (
	"testing"
	"time"

	"github.com/careeros/api/internal/opportunitytype"
)

func TestValidateVerifyRequest_OpenRequiresApplicationURL(t *testing.T) {
	err := ValidateVerifyRequest(VerifyRequest{
		ApplicationStatus:     opportunitytype.ApplicationStatusOpen,
		VerificationSourceURL: "https://example.edu/reu",
	}, time.Now())
	if err == nil {
		t.Fatal("expected error for missing application_url")
	}
}

func TestValidateVerifyRequest_OpenRejectsGenericETAP(t *testing.T) {
	url := "https://etap.nsf.gov"
	err := ValidateVerifyRequest(VerifyRequest{
		ApplicationStatus:     opportunitytype.ApplicationStatusOpen,
		ApplicationURL:        &url,
		VerificationSourceURL: "https://example.edu/reu",
	}, time.Now())
	if err == nil {
		t.Fatal("expected error for generic ETAP application_url")
	}
}

func TestValidateVerifyRequest_DeadlineBeforeOpensAt(t *testing.T) {
	opens := "2026-03-01"
	deadline := "2026-02-01"
	app := "https://example.edu/reu/apply"
	err := ValidateVerifyRequest(VerifyRequest{
		ApplicationStatus:     opportunitytype.ApplicationStatusOpen,
		ApplicationURL:        &app,
		VerificationSourceURL: "https://example.edu/reu",
		OpensAt:               &opens,
		Deadline:              &deadline,
	}, time.Now())
	if err == nil {
		t.Fatal("expected error when deadline before opens_at")
	}
}

func TestValidateVerifyRequest_UnknownRejectsApplicationURL(t *testing.T) {
	app := "https://example.edu/reu/apply"
	err := ValidateVerifyRequest(VerifyRequest{
		ApplicationStatus: opportunitytype.ApplicationStatusUnknown,
		ApplicationURL:    &app,
	}, time.Now())
	if err == nil {
		t.Fatal("expected error for application_url with unknown status")
	}
}

func TestValidateVerifyRequest_UnknownAllowsSourceURL(t *testing.T) {
	err := ValidateVerifyRequest(VerifyRequest{
		ApplicationStatus:     opportunitytype.ApplicationStatusUnknown,
		VerificationSourceURL: "https://example.edu/reu",
		VerificationMethod:    opportunitytype.AvailabilityMethodManualOfficialPage,
	}, time.Now())
	if err != nil {
		t.Fatalf("expected unknown with source URL to be valid: %v", err)
	}
}

func TestComputeNextVerification_OpenWithDeadline(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	next := ComputeNextVerification(opportunitytype.ApplicationStatusOpen, nil, &deadline, now)
	expected := now.AddDate(0, 0, 14)
	if !next.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, next)
	}
}
