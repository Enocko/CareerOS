package notifications

import "errors"

// ErrNotFound is returned when a notification does not exist for the user.
var ErrNotFound = errors.New("notification not found")

const (
	TypeApplicationDeadline = "application_deadline"
	TypeSavedDeadline       = "saved_opportunity_deadline"
)
