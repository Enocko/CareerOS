package applications

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// ValidateCreateRequest validates application creation input.
func ValidateCreateRequest(req CreateRequest) error {
	if req.OpportunityID == uuid.Nil {
		return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Validation failed").
			WithDetails([]platform.FieldError{{Field: "opportunity_id", Message: "is required"}})
	}
	return nil
}

// ValidateUpdateRequest validates application update input.
func ValidateUpdateRequest(req UpdateRequest) error {
	hasField := req.CurrentStatus != nil || req.Notes != nil ||
		req.NextAction != nil || req.NextActionDate != nil || req.InterviewDate != nil

	if !hasField {
		return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "At least one field must be provided")
	}

	if req.CurrentStatus != nil {
		status := strings.TrimSpace(*req.CurrentStatus)
		if !ValidStatuses[status] {
			return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Validation failed").
				WithDetails([]platform.FieldError{{
					Field:   "current_status",
					Message: fmt.Sprintf("must be one of: %s", validStatusList()),
				}})
		}
	}

	return nil
}

func validStatusList() string {
	statuses := []string{
		"saved", "preparing", "applied", "oa_assessment", "interview",
		"final_round", "offer", "rejected", "withdrawn", "closed",
	}
	return strings.Join(statuses, ", ")
}
