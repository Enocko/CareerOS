package applications

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Service handles application business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new applications Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new application for the authenticated student.
func (s *Service) Create(ctx context.Context, studentID uuid.UUID, req CreateRequest) (*Application, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	app, err := s.repo.Create(ctx, studentID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicate):
			return nil, platform.NewAppError(http.StatusConflict, platform.ErrorCodeConflict, "Application already exists for this opportunity")
		case errors.Is(err, ErrOpportunityNotFound):
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found")
		case errors.Is(err, ErrOpportunityClosed):
			return nil, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Opportunity is not open")
		case errors.Is(err, ErrOpportunityNotEmployment):
			return nil, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Applications are only supported for employment opportunities")
		default:
			return nil, platform.InternalError()
		}
	}
	return app, nil
}

// GetByID returns an application with status history.
func (s *Service) GetByID(ctx context.Context, studentID, applicationID uuid.UUID) (*Application, error) {
	app, err := s.repo.GetByID(ctx, studentID, applicationID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found")
		}
		return nil, platform.InternalError()
	}
	return app, nil
}

// List returns paginated applications for the dashboard.
func (s *Service) List(ctx context.Context, studentID uuid.UUID, filter ListFilter) (*platform.PaginatedResponse[Application], error) {
	filter.Page, filter.PerPage = platform.ParsePagination(filter.Page, filter.PerPage)

	if filter.Status != "" && !ValidStatuses[filter.Status] {
		return nil, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid status filter")
	}

	results, total, err := s.repo.List(ctx, studentID, filter)
	if err != nil {
		return nil, platform.InternalError()
	}

	return &platform.PaginatedResponse[Application]{
		Data:       results,
		Pagination: platform.NewPagination(filter.Page, filter.PerPage, total),
	}, nil
}

// Update updates an application.
func (s *Service) Update(ctx context.Context, studentID, applicationID uuid.UUID, req UpdateRequest) (*Application, error) {
	if err := ValidateUpdateRequest(req); err != nil {
		return nil, err
	}

	app, err := s.repo.Update(ctx, studentID, applicationID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found")
		}
		return nil, platform.InternalError()
	}
	return app, nil
}

// Delete removes an application the student has not yet submitted.
func (s *Service) Delete(ctx context.Context, studentID, applicationID uuid.UUID) error {
	err := s.repo.Delete(ctx, studentID, applicationID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found")
		case errors.Is(err, ErrCannotRemove):
			return platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation,
				"Only applications you have not submitted yet can be removed. Use status \"withdrawn\" instead.")
		default:
			return platform.InternalError()
		}
	}
	return nil
}

// ParseListFilter extracts list filter from query values.
func ParseListFilter(values map[string][]string) ListFilter {
	get := func(key string) string {
		if v, ok := values[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	page, _ := strconv.Atoi(get("page"))
	perPage, _ := strconv.Atoi(get("per_page"))
	return ListFilter{
		Status:  get("status"),
		Page:    page,
		PerPage: perPage,
	}
}
