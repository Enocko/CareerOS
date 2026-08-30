package opportunityreports

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/platform"
)

const maxNoteLen = 500

// ValidateCreateRequest validates a student report payload.
func ValidateCreateRequest(req CreateRequest) error {
	if !ValidReasons[req.Reason] {
		return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid report reason")
	}
	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		if len(note) > maxNoteLen {
			return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Note must be 500 characters or fewer")
		}
	}
	return nil
}

// ValidateUpdateStatusRequest validates admin triage status.
func ValidateUpdateStatusRequest(req UpdateStatusRequest) error {
	if req.Status != StatusResolved && req.Status != StatusDismissed {
		return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Status must be resolved or dismissed")
	}
	return nil
}
