package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Service handles student profile business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new profile Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetProfile returns the profile for the authenticated user.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	profile, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Profile not found")
		}
		return nil, platform.InternalError()
	}
	return profile, nil
}

// UpsertProfile creates or updates the profile for the authenticated user.
func (s *Service) UpsertProfile(ctx context.Context, userID uuid.UUID, req UpdateRequest) (*Profile, error) {
	if err := ValidateUpdateRequest(req); err != nil {
		return nil, err
	}

	profile, err := s.repo.Upsert(ctx, userID, req)
	if err != nil {
		return nil, platform.InternalError()
	}
	return profile, nil
}
