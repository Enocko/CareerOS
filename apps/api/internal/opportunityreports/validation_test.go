package opportunityreports_test

import (
	"testing"

	"github.com/careeros/api/internal/opportunityreports"
)

func TestValidateCreateRequest(t *testing.T) {
	if err := opportunityreports.ValidateCreateRequest(opportunityreports.CreateRequest{
		Reason: opportunityreports.ReasonBrokenLink,
	}); err != nil {
		t.Fatalf("expected valid reason: %v", err)
	}
	if err := opportunityreports.ValidateCreateRequest(opportunityreports.CreateRequest{
		Reason: "invalid",
	}); err == nil {
		t.Fatal("expected invalid reason error")
	}
	note := string(make([]byte, 501))
	if err := opportunityreports.ValidateCreateRequest(opportunityreports.CreateRequest{
		Reason: opportunityreports.ReasonOther,
		Note:   &note,
	}); err == nil {
		t.Fatal("expected note length error")
	}
}

func TestValidateUpdateStatusRequest(t *testing.T) {
	if err := opportunityreports.ValidateUpdateStatusRequest(opportunityreports.UpdateStatusRequest{
		Status: opportunityreports.StatusResolved,
	}); err != nil {
		t.Fatalf("expected valid status: %v", err)
	}
	if err := opportunityreports.ValidateUpdateStatusRequest(opportunityreports.UpdateStatusRequest{
		Status: "pending",
	}); err == nil {
		t.Fatal("expected invalid admin status error")
	}
}
