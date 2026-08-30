package researchverification

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/careeros/api/internal/opportunitytype"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
	"net/http"
)

// Service handles research availability verification workflow.
type Service struct {
	repo *Repository
}

// NewService creates a research verification service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListQueue returns prioritized verification candidates.
func (s *Service) ListQueue(ctx context.Context, limit int) ([]QueueItem, error) {
	return s.repo.ListQueue(ctx, limit)
}

// GetOpportunity returns a research opportunity for review.
func (s *Service) GetOpportunity(ctx context.Context, id uuid.UUID) (*QueueItem, error) {
	item, err := s.repo.GetResearchOpportunity(ctx, id)
	if err != nil {
		return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found")
	}
	return item, nil
}

// ListVerifications returns verification history.
func (s *Service) ListVerifications(ctx context.Context, opportunityID uuid.UUID, limit int) ([]VerificationRecord, error) {
	if _, err := s.repo.GetResearchOpportunity(ctx, opportunityID); err != nil {
		return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found")
	}
	return s.repo.ListVerifications(ctx, opportunityID, limit)
}

// Verify applies a new availability verification.
func (s *Service) Verify(ctx context.Context, opportunityID, reviewerID uuid.UUID, req VerifyRequest) (*VerificationRecord, error) {
	now := time.Now().UTC()
	if err := ValidateVerifyRequest(req, now); err != nil {
		return nil, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, err.Error())
	}

	item, err := s.repo.GetResearchOpportunity(ctx, opportunityID)
	if err != nil {
		return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found")
	}

	status := strings.TrimSpace(req.ApplicationStatus)
	method := strings.TrimSpace(req.VerificationMethod)
	if method == "" {
		method = opportunitytype.AvailabilityMethodManualOfficialPage
	}

	var opensAt, deadline *time.Time
	if req.OpensAt != nil && strings.TrimSpace(*req.OpensAt) != "" {
		t, _ := time.Parse("2006-01-02", strings.TrimSpace(*req.OpensAt))
		opensAt = &t
	}
	if req.Deadline != nil && strings.TrimSpace(*req.Deadline) != "" {
		t, _ := time.Parse("2006-01-02", strings.TrimSpace(*req.Deadline))
		deadline = &t
	}

	nextAt := ComputeNextVerification(status, opensAt, deadline, now)

	var appURL *string
	if req.ApplicationURL != nil && strings.TrimSpace(*req.ApplicationURL) != "" {
		v := strings.TrimSpace(*req.ApplicationURL)
		appURL = &v
	}

	var oppDeadline *time.Time
	if status == opportunitytype.ApplicationStatusOpen || status == opportunitytype.ApplicationStatusUpcoming {
		oppDeadline = deadline
	}

	cycleLabel := ""
	if req.CycleLabel != nil {
		cycleLabel = strings.TrimSpace(*req.CycleLabel)
	}
	opensAtStr := ""
	if opensAt != nil {
		opensAtStr = opensAt.Format("2006-01-02")
	}

	sourceURL := strings.TrimSpace(req.VerificationSourceURL)
	var sourceURLPtr *string
	if sourceURL != "" {
		sourceURLPtr = &sourceURL
	}

	prevStatus := item.ApplicationStatus
	meta, err := BuildTypeMetadata(
		item.TypeMetadata,
		status,
		method,
		sourceURL,
		cycleLabel,
		opensAtStr,
		now.Format(time.RFC3339),
		nextAt.Format(time.RFC3339),
		item.ProgramURL,
	)
	if err != nil {
		return nil, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, err.Error())
	}

	rec, err := s.repo.ApplyVerification(ctx, applyVerificationInput{
		OpportunityID:         opportunityID,
		ApplicationStatus:     status,
		ApplicationURL:        appURL,
		VerificationSourceURL: sourceURLPtr,
		OpensAt:               opensAt,
		Deadline:              deadline,
		CycleLabel:            req.CycleLabel,
		VerificationMethod:    method,
		VerifiedBy:            reviewerID,
		NextVerificationAt:    nextAt,
		Notes:                 req.Notes,
		TypeMetadata:          meta,
		OpportunityDeadline:   oppDeadline,
		Now:                   now,
	})
	if err != nil {
		return nil, platform.InternalError()
	}

	slog.Info("research availability verification applied",
		"opportunity_id", opportunityID,
		"previous_status", prevStatus,
		"new_status", status,
		"verification_method", method,
		"verified_at", now,
		"verified_by", reviewerID,
	)

	return rec, nil
}

// GetMetrics returns catalog metrics.
func (s *Service) GetMetrics(ctx context.Context) (Metrics, error) {
	return s.repo.GetMetrics(ctx)
}
