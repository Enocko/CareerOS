package auth

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/careeros/api/internal/platform"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	maxEmailLength    = 255
)

// ValidateRegisterRequest validates registration input.
func ValidateRegisterRequest(req RegisterRequest) error {
	return validateCredentials(req.Email, req.Password)
}

// ValidateLoginRequest validates login input.
func ValidateLoginRequest(req LoginRequest) error {
	return validateCredentials(req.Email, req.Password)
}

func validateCredentials(email, password string) error {
	var details []platform.FieldError

	email = strings.TrimSpace(email)
	if email == "" {
		details = append(details, platform.FieldError{Field: "email", Message: "is required"})
	} else if len(email) > maxEmailLength {
		details = append(details, platform.FieldError{Field: "email", Message: fmt.Sprintf("must be at most %d characters", maxEmailLength)})
	} else if _, err := mail.ParseAddress(email); err != nil {
		details = append(details, platform.FieldError{Field: "email", Message: "must be a valid email address"})
	}

	if password == "" {
		details = append(details, platform.FieldError{Field: "password", Message: "is required"})
	} else if len(password) < minPasswordLength {
		details = append(details, platform.FieldError{Field: "password", Message: fmt.Sprintf("must be at least %d characters", minPasswordLength)})
	} else if len(password) > maxPasswordLength {
		details = append(details, platform.FieldError{Field: "password", Message: fmt.Sprintf("must be at most %d characters", maxPasswordLength)})
	}

	if len(details) > 0 {
		return platform.NewAppError(400, platform.ErrorCodeValidation, "Validation failed").WithDetails(details)
	}

	return nil
}

// NormalizeEmail lowercases and trims an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
