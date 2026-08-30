package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// ProfileEnsurer creates a default student profile after registration.
type ProfileEnsurer interface {
	EnsureDefaultProfile(ctx context.Context, userID uuid.UUID) error
}

// Service handles authentication business logic.
type Service struct {
	repo     *Repository
	tokens   *TokenManager
	profiles ProfileEnsurer
}

// NewService creates a new auth Service.
func NewService(repo *Repository, tokens *TokenManager, profiles ProfileEnsurer) *Service {
	return &Service{repo: repo, tokens: tokens, profiles: profiles}
}

// Register creates a new user account and returns an auth response.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := ValidateRegisterRequest(req); err != nil {
		return nil, err
	}

	email := NormalizeEmail(req.Email)

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, platform.InternalError()
	}

	user, err := s.repo.CreateUser(ctx, email, hash)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return nil, platform.NewAppError(http.StatusConflict, platform.ErrorCodeConflict, "Email already registered")
		}
		return nil, platform.InternalError()
	}

	if s.profiles != nil {
		if err := s.profiles.EnsureDefaultProfile(ctx, user.ID); err != nil {
			return nil, platform.InternalError()
		}
	}

	token, err := s.tokens.Generate(user)
	if err != nil {
		return nil, platform.InternalError()
	}

	resp := user.ToResponse()
	return &AuthResponse{User: resp, Token: token}, nil
}

// Login authenticates a user and returns an auth response.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if err := ValidateLoginRequest(req); err != nil {
		return nil, err
	}

	email := NormalizeEmail(req.Email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Invalid email or password")
		}
		return nil, platform.InternalError()
	}

	if !CheckPassword(user.PasswordHash, req.Password) {
		return nil, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Invalid email or password")
	}

	if s.profiles != nil {
		if err := s.profiles.EnsureDefaultProfile(ctx, user.ID); err != nil {
			return nil, platform.InternalError()
		}
	}

	token, err := s.tokens.Generate(user)
	if err != nil {
		return nil, platform.InternalError()
	}

	resp := user.ToResponse()
	return &AuthResponse{User: resp, Token: token}, nil
}

// GetUser returns the authenticated user's public profile.
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "User not found")
		}
		return nil, platform.InternalError()
	}

	resp := user.ToResponse()
	return &resp, nil
}
