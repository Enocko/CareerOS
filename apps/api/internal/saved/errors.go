package saved

import "errors"

var (
	ErrOpportunityNotFound = errors.New("opportunity not found")
	ErrSaveNotFound        = errors.New("save not found")
)
