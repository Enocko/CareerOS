package opportunityreports

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Service handles opportunity report business logic.
type Service struct {
	repo *Repository
}

// NewService creates a report service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create records a student issue report without changing listing status.
func (s *Service) Create(ctx context.Context, opportunityID, reporterID uuid.UUID, req CreateRequest) (*Report, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return nil, err
	}
	exists, err := s.repo.OpportunityExists(ctx, opportunityID)
	if err != nil {
		return nil, platform.InternalError()
	}
	if !exists {
		return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found")
	}
	var note *string
	if req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		if trimmed != "" {
			note = &trimmed
		}
	}
	rep, err := s.repo.Create(ctx, opportunityID, reporterID, req.Reason, note)
	if err != nil {
		return nil, platform.InternalError()
	}
	return rep, nil
}

// ListQueue returns grouped pending reports for operators.
func (s *Service) ListQueue(ctx context.Context, limit int) ([]QueueItem, error) {
	items, err := s.repo.ListQueue(ctx, limit)
	if err != nil {
		return nil, platform.InternalError()
	}
	return items, nil
}

// CountPending returns pending report count.
func (s *Service) CountPending(ctx context.Context) (int, error) {
	count, err := s.repo.CountPending(ctx)
	if err != nil {
		return 0, platform.InternalError()
	}
	return count, nil
}

// ResolveOpportunity marks all pending reports for an opportunity resolved or dismissed.
func (s *Service) ResolveOpportunity(ctx context.Context, opportunityID, resolverID uuid.UUID, req UpdateStatusRequest) (int, error) {
	if err := ValidateUpdateStatusRequest(req); err != nil {
		return 0, err
	}
	exists, err := s.repo.OpportunityExists(ctx, opportunityID)
	if err != nil {
		return 0, platform.InternalError()
	}
	if !exists {
		return 0, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found")
	}
	updated, err := s.repo.UpdateOpportunityReportsStatus(ctx, opportunityID, req.Status, resolverID)
	if err != nil {
		return 0, platform.InternalError()
	}
	if updated == 0 {
		return 0, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "No pending reports for this opportunity")
	}
	return updated, nil
}

// GetByID returns a report or not found.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	rep, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Report not found")
		}
		return nil, platform.InternalError()
	}
	return rep, nil
}
