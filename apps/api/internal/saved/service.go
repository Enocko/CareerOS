package saved

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Service handles saved opportunity business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new saved Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Save bookmarks an opportunity for a student.
func (s *Service) Save(ctx context.Context, studentID, opportunityID uuid.UUID) (*SavedOpportunity, error) {
	saved, err := s.repo.Save(ctx, studentID, opportunityID)
	if err != nil {
		if errors.Is(err, ErrOpportunityNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found")
		}
		return nil, platform.InternalError()
	}
	return saved, nil
}

// Unsave removes a bookmark.
func (s *Service) Unsave(ctx context.Context, studentID, opportunityID uuid.UUID) error {
	err := s.repo.Unsave(ctx, studentID, opportunityID)
	if err != nil {
		if errors.Is(err, ErrSaveNotFound) {
			return platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Save not found")
		}
		return platform.InternalError()
	}
	return nil
}

// List returns paginated saved opportunities.
func (s *Service) List(ctx context.Context, studentID uuid.UUID, filter ListFilter) (*platform.PaginatedResponse[Summary], error) {
	filter.Page, filter.PerPage = platform.ParsePagination(filter.Page, filter.PerPage)

	results, total, err := s.repo.List(ctx, studentID, filter)
	if err != nil {
		return nil, platform.InternalError()
	}

	return &platform.PaginatedResponse[Summary]{
		Data:       results,
		Pagination: platform.NewPagination(filter.Page, filter.PerPage, total),
	}, nil
}

// ParseListFilter extracts pagination from query values.
func ParseListFilter(values map[string][]string) ListFilter {
	get := func(key string) string {
		if v, ok := values[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	page, _ := strconv.Atoi(get("page"))
	perPage, _ := strconv.Atoi(get("per_page"))
	return ListFilter{Page: page, PerPage: perPage}
}
