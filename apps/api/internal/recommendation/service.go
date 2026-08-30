package recommendation

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/careeros/api/internal/opportunities"
	"github.com/careeros/api/internal/platform"
	"github.com/careeros/api/internal/profile"
	"github.com/google/uuid"
)

// Service orchestrates personalized opportunity recommendations.
type Service struct {
	repo        *Repository
	profileRepo *profile.Repository
}

// NewService creates a recommendation service.
func NewService(repo *Repository, profileRepo *profile.Repository) *Service {
	return &Service{repo: repo, profileRepo: profileRepo}
}

// Recommend returns ranked opportunities for a student.
func (s *Service) Recommend(ctx context.Context, studentID uuid.UUID, filter ListFilter) (*platform.PaginatedResponse[Result], *RecommendMeta, error) {
	filter.Page, filter.PerPage = platform.ParsePagination(filter.Page, filter.PerPage)
	now := time.Now().UTC()

	candidates, err := s.repo.ListCandidates(ctx, studentID)
	if err != nil {
		return nil, nil, err
	}

	var studentCtx StudentContext
	prof, err := s.profileRepo.GetByUserID(ctx, studentID)
	if err != nil && !errors.Is(err, profile.ErrProfileNotFound) {
		return nil, nil, err
	}
	studentCtx = StudentContextFromProfile(prof)

	scored := make([]Result, 0, len(candidates))
	for _, c := range candidates {
		if !PassesHardFilters(c, now) {
			continue
		}
		score, factors := Score(studentCtx, c, now)
		reasons, summary := BuildReasons(studentCtx, c, factors)
		scored = append(scored, Result{
			Opportunity: candidateToSummary(c),
			MatchScore:  score,
			Factors:     factors,
			Reasons:     reasons,
			ReasonShort: summary,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].MatchScore != scored[j].MatchScore {
			return scored[i].MatchScore > scored[j].MatchScore
		}
		return scored[i].Opportunity.Title < scored[j].Opportunity.Title
	})

	total := len(scored)
	start := (filter.Page - 1) * filter.PerPage
	if start > total {
		start = total
	}
	end := start + filter.PerPage
	if end > total {
		end = total
	}
	pageData := scored[start:end]
	if pageData == nil {
		pageData = []Result{}
	}

	meta := &RecommendMeta{
		ProfileComplete: studentCtx.ProfileComplete,
		HasProfile:      studentCtx.HasProfile,
		CatalogScored:   len(candidates),
		EligibleCount:   total,
	}

	return &platform.PaginatedResponse[Result]{
		Data:       pageData,
		Pagination: platform.NewPagination(filter.Page, filter.PerPage, total),
	}, meta, nil
}

// RecommendMeta carries non-paginated recommendation context.
type RecommendMeta struct {
	ProfileComplete bool `json:"profile_complete"`
	HasProfile      bool `json:"has_profile"`
	CatalogScored   int  `json:"catalog_scored"`
	EligibleCount   int  `json:"eligible_count"`
}

func candidateToSummary(c Candidate) opportunities.Summary {
	return opportunities.Summary{
		ID:                 c.ID,
		Title:              c.Title,
		OrganizationName:   c.OrganizationName,
		Category:           c.Category,
		Location:           c.Location,
		WorkArrangement:    c.WorkArrangement,
		Deadline:           c.Deadline,
		Skills:             c.Skills,
		Tags:               c.Tags,
		Status:             c.Status,
		VerificationStatus: c.VerificationStatus,
		SourceName:         c.SourceName,
		LastCheckedAt:      c.LastCheckedAt,
		ExperienceLevel:    c.ExperienceLevel,
		CareerFamily:       c.CareerFamily,
		RelevanceTier:      c.RelevanceTier,
		IsSaved:            c.IsSaved,
	}
}
