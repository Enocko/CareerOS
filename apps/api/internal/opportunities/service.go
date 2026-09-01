package opportunities

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/careeros/api/internal/opportunitytype"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Service handles opportunity business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new opportunities Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns a paginated list of opportunities for the authenticated student.
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

// GetByID returns a single opportunity for the authenticated student.
func (s *Service) GetByID(ctx context.Context, studentID, opportunityID uuid.UUID) (*Opportunity, error) {
	opp, err := s.repo.GetByID(ctx, studentID, opportunityID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found")
		}
		return nil, platform.InternalError()
	}
	return opp, nil
}

// ParseListFilter extracts list filter parameters from query values.
func ParseListFilter(values map[string][]string) ListFilter {
	get := func(key string) string {
		if v, ok := values[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}

	page, _ := strconv.Atoi(get("page"))
	perPage, _ := strconv.Atoi(get("per_page"))
	includeUnverified := get("include_unverified") == "true" || get("include_unverified") == "1"
	includeAmbiguous := get("include_ambiguous") == "true" || get("include_ambiguous") == "1"
	includeNonTechnical := get("include_non_technical") == "true" || get("include_non_technical") == "1"

	scope := get("type")
	if scope == "" {
		if ot := get("opportunity_type"); ot == opportunitytype.Research {
			scope = CatalogScopeResearch
		} else if ot != "" {
			scope = CatalogScopeEmployment
		} else {
			scope = CatalogScopeEmployment
		}
	}

	filter := ListFilter{
		Query:               get("q"),
		Category:            get("category"),
		ApplicationStatus:   get("application_status"),
		WorkArrangement:     get("work_arrangement"),
		Location:            get("location"),
		CareerFamily:        get("career_family"),
		ExperienceLevel:     get("experience_level"),
		IncludeUnverified:   includeUnverified,
		IncludeAmbiguous:    includeAmbiguous,
		IncludeNonTechnical: includeNonTechnical,
		CatalogScope:        scope,
		Sort:                normalizeSort(get("sort"), scope),
		Page:                page,
		PerPage:             perPage,
	}
	switch scope {
	case CatalogScopeResearch:
		filter.OpportunityType = opportunitytype.Research
	case CatalogScopeEmployment:
		filter.OpportunityType = opportunitytype.Employment
	}
	return filter
}

func normalizeSort(raw, scope string) string {
	switch raw {
	case SortDeadline, SortArrangement:
		return raw
	case SortNewest:
		return SortNewest
	default:
		if scope == CatalogScopeResearch {
			return ""
		}
		return SortNewest
	}
}
