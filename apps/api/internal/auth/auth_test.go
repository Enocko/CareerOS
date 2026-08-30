package auth

import (
	"testing"
	"time"

	"github.com/careeros/api/internal/config"
	"github.com/google/uuid"
)

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("securepass123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if !CheckPassword(hash, "securepass123") {
		t.Error("expected password to match hash")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("expected wrong password to not match")
	}
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{"valid", RegisterRequest{Email: "student@gram.edu", Password: "securepass123"}, false},
		{"missing email", RegisterRequest{Password: "securepass123"}, true},
		{"invalid email", RegisterRequest{Email: "not-an-email", Password: "securepass123"}, true},
		{"short password", RegisterRequest{Email: "student@gram.edu", Password: "short"}, true},
		{"missing password", RegisterRequest{Email: "student@gram.edu"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegisterRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Student@Gram.EDU  "); got != "student@gram.edu" {
		t.Errorf("expected student@gram.edu, got %s", got)
	}
}

func TestTokenManager_GenerateAndValidate(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:      "test-secret-at-least-16-chars",
		JWTExpiryHours: 24,
	}
	tm := NewTokenManager(cfg)

	user := &User{
		ID:    uuid.New(),
		Email: "student@gram.edu",
	}

	token, err := tm.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := tm.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, claims.Email)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:      "test-secret-at-least-16-chars",
		JWTExpiryHours: -1,
	}
	tm := NewTokenManager(cfg)

	user := &User{ID: uuid.New(), Email: "student@gram.edu"}
	token, err := tm.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = tm.Validate(token)
	if err == nil {
		t.Error("expected expired token to fail validation")
	}
}
