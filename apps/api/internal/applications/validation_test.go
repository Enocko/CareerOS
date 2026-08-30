package applications

import (
	"testing"
)

func strPtr(v string) *string { return &v }

func TestValidateCreateRequest(t *testing.T) {
	err := ValidateCreateRequest(CreateRequest{})
	if err == nil {
		t.Error("expected error for missing opportunity_id")
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	err := ValidateUpdateRequest(UpdateRequest{})
	if err == nil {
		t.Error("expected error for empty update")
	}

	err = ValidateUpdateRequest(UpdateRequest{CurrentStatus: strPtr("invalid")})
	if err == nil {
		t.Error("expected error for invalid status")
	}

	err = ValidateUpdateRequest(UpdateRequest{CurrentStatus: strPtr("applied")})
	if err != nil {
		t.Errorf("expected valid update, got %v", err)
	}
}
