package platform

import "net/http"

// ErrorCode represents a machine-readable API error code.
type ErrorCode string

const (
	ErrorCodeValidation  ErrorCode = "VALIDATION_ERROR"
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden   ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrorCodeConflict    ErrorCode = "CONFLICT"
	ErrorCodeInternal    ErrorCode = "INTERNAL_ERROR"
)

// FieldError describes a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// APIError is the standard error envelope returned by the API.
type APIError struct {
	Code    ErrorCode    `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// AppError wraps an API error with an HTTP status code.
type AppError struct {
	Status  int
	Code    ErrorCode
	Message string
	Details []FieldError
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError creates an AppError with the given status and code.
func NewAppError(status int, code ErrorCode, message string) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

// WithDetails adds field-level validation details to an AppError.
func (e *AppError) WithDetails(details []FieldError) *AppError {
	e.Details = details
	return e
}

// InternalError returns a generic 500 error without leaking internals.
func InternalError() *AppError {
	return NewAppError(http.StatusInternalServerError, ErrorCodeInternal, "An unexpected error occurred")
}
