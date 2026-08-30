package applications

import "errors"

var (
	ErrNotFound            = errors.New("application not found")
	ErrForbidden           = errors.New("application forbidden")
	ErrDuplicate           = errors.New("application already exists")
	ErrOpportunityNotFound = errors.New("opportunity not found")
	ErrOpportunityClosed     = errors.New("opportunity is closed")
	ErrOpportunityNotEmployment = errors.New("opportunity is not employment")
	ErrCannotRemove        = errors.New("application cannot be removed")
)

// RemovableStatuses are pipeline stages where tracking can be deleted entirely.
var RemovableStatuses = map[string]bool{
	"saved":      true,
	"preparing":  true,
}
