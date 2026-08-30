package opportunities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles opportunity persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new opportunities Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List returns a paginated list of open opportunities matching the filter.
func (r *Repository) List(ctx context.Context, studentID uuid.UUID, filter ListFilter) ([]Summary, int, error) {
	where, filterArgs := buildListWhere(filter, 1)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM opportunities o
		WHERE %s
	`, where)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count opportunities: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	listWhere, listFilterArgs := buildListWhere(filter, 2)
	listArgs := append([]any{studentID}, listFilterArgs...)
	listArgs = append(listArgs, filter.PerPage, offset)

	limitArg := fmt.Sprintf("$%d", len(listArgs)-1)
	offsetArg := fmt.Sprintf("$%d", len(listArgs))

	orderBy := "o.deadline ASC NULLS LAST, o.created_at DESC"
	switch filter.CatalogScope {
	case CatalogScopeAll:
		orderBy = `CASE
			WHEN o.opportunity_type = 'research' AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'open' THEN 1
			WHEN o.opportunity_type = 'research' AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN o.opportunity_type = 'employment' THEN 3
			WHEN o.opportunity_type = 'research' AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN o.opportunity_type = 'research' AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END, o.deadline ASC NULLS LAST, o.created_at DESC`
	case CatalogScopeResearch:
		orderBy = researchOrderBy()
	}

	listQuery := fmt.Sprintf(`
		SELECT o.id, o.title, o.organization_name, o.category, o.opportunity_type,
		       o.verification_method, o.employment_mode, o.location,
		       o.work_arrangement, o.deadline, o.skills, o.tags, o.status,
		       o.verification_status, o.source, o.last_checked_at,
		       o.experience_level, o.career_family, o.relevance_tier,
		       o.type_metadata,
		       (so.id IS NOT NULL) AS is_saved
		FROM opportunities o
		LEFT JOIN saved_opportunities so
			ON so.opportunity_id = o.id AND so.student_id = $1
		WHERE %s
		ORDER BY %s
		LIMIT %s OFFSET %s
	`, listWhere, orderBy, limitArg, offsetArg)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list opportunities: %w", err)
	}
	defer rows.Close()

	var results []Summary
	for rows.Next() {
		var s Summary
		var typeMetadata json.RawMessage
		if err := rows.Scan(
			&s.ID, &s.Title, &s.OrganizationName, &s.Category, &s.OpportunityType,
			&s.VerificationMethod, &s.EmploymentMode, &s.Location,
			&s.WorkArrangement, &s.Deadline, &s.Skills, &s.Tags, &s.Status,
			&s.VerificationStatus, &s.SourceName, &s.LastCheckedAt,
			&s.ExperienceLevel, &s.CareerFamily, &s.RelevanceTier,
			&typeMetadata, &s.IsSaved,
		); err != nil {
			return nil, 0, fmt.Errorf("scan opportunity: %w", err)
		}
		if len(typeMetadata) > 0 && string(typeMetadata) != "{}" {
			s.TypeMetadata = typeMetadata
		}
		s.Skills = normalizeSlice(s.Skills)
		s.Tags = normalizeSlice(s.Tags)
		results = append(results, s)
	}

	if results == nil {
		results = []Summary{}
	}

	return results, total, rows.Err()
}

// GetByID returns a single opportunity with student-specific flags.
// Non-discoverable listings (closed, stale, etc.) are only returned when the
// student previously saved, tracked, or viewed the opportunity.
func (r *Repository) GetByID(ctx context.Context, studentID, opportunityID uuid.UUID) (*Opportunity, error) {
	var o Opportunity
	var sourceName string
	var sourceURL *string
	var externalID *string
	var classificationReasons []string
	var hasViewed bool
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.title, o.organization_name, o.description, o.category,
		       o.opportunity_type, o.type_metadata, o.verification_method, o.employment_mode,
		       o.location, o.work_arrangement, o.deadline, o.start_date, o.eligibility,
		       o.skills, o.compensation, o.application_url, o.source, o.source_url,
		       o.verification_status, o.last_checked_at, o.last_seen_at, o.status, o.tags,
		       o.experience_level, o.career_family, o.education_level, o.relevance_tier,
		       o.classification_reasons, o.external_id,
		       o.created_at, o.updated_at,
		       EXISTS(
		           SELECT 1 FROM saved_opportunities so
		           WHERE so.opportunity_id = o.id AND so.student_id = $1
		       ) AS is_saved,
		       EXISTS(
		           SELECT 1 FROM applications a
		           WHERE a.opportunity_id = o.id AND a.student_id = $1
		       ) AS has_application,
		       EXISTS(
		           SELECT 1 FROM opportunity_views ov
		           WHERE ov.opportunity_id = o.id AND ov.student_id = $1
		       ) AS has_viewed
		FROM opportunities o
		WHERE o.id = $2
	`, studentID, opportunityID).Scan(
		&o.ID, &o.Title, &o.OrganizationName, &o.Description, &o.Category,
		&o.OpportunityType, &o.TypeMetadata, &o.VerificationMethod, &o.EmploymentMode,
		&o.Location, &o.WorkArrangement, &o.Deadline, &o.StartDate, &o.Eligibility,
		&o.Skills, &o.Compensation, &o.ApplicationURL, &sourceName, &sourceURL,
		&o.VerificationStatus, &o.LastCheckedAt, &o.LastSeenAt, &o.Status, &o.Tags,
		&o.ExperienceLevel, &o.CareerFamily, &o.EducationLevel, &o.RelevanceTier,
		&classificationReasons, &externalID,
		&o.CreatedAt, &o.UpdatedAt, &o.IsSaved, &o.HasApplication, &hasViewed,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get opportunity: %w", err)
	}

	extID := ""
	if externalID != nil {
		extID = *externalID
	}
	if !canAccessDetail(&o, extID, hasViewed) {
		return nil, ErrNotFound
	}

	if err := r.RecordView(ctx, studentID, opportunityID); err != nil {
		return nil, fmt.Errorf("record opportunity view: %w", err)
	}

	o.Source = SourceAttribution{Name: sourceName, SourceURL: sourceURL}
	o.Skills = normalizeSlice(o.Skills)
	o.Tags = normalizeSlice(o.Tags)
	o.ClassificationReasons = normalizeSlice(classificationReasons)
	return &o, nil
}

// RecordView upserts the student's most recent detail view timestamp.
func (r *Repository) RecordView(ctx context.Context, studentID, opportunityID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO opportunity_views (student_id, opportunity_id, last_viewed_at)
		VALUES ($1, $2, now())
		ON CONFLICT (student_id, opportunity_id)
		DO UPDATE SET last_viewed_at = EXCLUDED.last_viewed_at
	`, studentID, opportunityID)
	if err != nil {
		return fmt.Errorf("upsert opportunity view: %w", err)
	}
	return nil
}

// Exists checks whether an open opportunity exists by ID.
func (r *Repository) Exists(ctx context.Context, opportunityID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM opportunities WHERE id = $1)
	`, opportunityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check opportunity exists: %w", err)
	}
	return exists, nil
}

func buildListWhere(filter ListFilter, startIndex int) (string, []any) {
	conditions := []string{"o.status = 'open'"}
	args := []any{}
	argIndex := startIndex

	if !filter.IncludeUnverified {
		conditions = append(conditions, "o.verification_status = 'verified'")
	} else {
		conditions = append(conditions, "o.verification_status IN ('verified', 'unverified')")
	}

	// Exclude automated test fixtures from student-facing catalog.
	conditions = append(conditions, `(
		o.external_id IS NULL OR (
			o.external_id NOT LIKE 'API-TEST-%' AND
			o.external_id NOT LIKE 'UPSERT-TEST-%' AND
			o.external_id NOT LIKE 'FAIL-SYNC-TEST-%' AND
			o.external_id NOT LIKE 'STALE-SYNC-TEST-%' AND
			o.external_id NOT LIKE 'GH-FAIL-BOARD-%' AND
			o.external_id NOT LIKE 'GH-OK-BOARD-%' AND
			o.external_id NOT LIKE 'ASHBY-FAIL-BOARD-%' AND
			o.external_id NOT LIKE 'ASHBY-OK-BOARD-%' AND
			o.external_id NOT LIKE 'LEVER-FAIL-BOARD-%' AND
			o.external_id NOT LIKE 'LEVER-OK-BOARD-%' AND
			o.external_id NOT LIKE 'REU-UPSERT-TEST-%'
		)
	)`)

	if filter.Query != "" {
		pattern := "%" + filter.Query + "%"
		conditions = append(conditions, fmt.Sprintf(
			`(o.title ILIKE $%d OR o.organization_name ILIKE $%d OR o.description ILIKE $%d
			  OR o.type_metadata->>'research_area' ILIKE $%d OR o.type_metadata->>'cycle_label' ILIKE $%d
			  OR EXISTS (SELECT 1 FROM unnest(o.skills) s WHERE s ILIKE $%d)
			  OR EXISTS (SELECT 1 FROM unnest(o.tags) t WHERE t ILIKE $%d))`,
			argIndex, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex,
		))
		args = append(args, pattern)
		argIndex++
	}

	if filter.Category != "" {
		if filter.CatalogScope == CatalogScopeAll {
			conditions = append(conditions, fmt.Sprintf(
				"(o.opportunity_type = 'research' OR o.category = $%d)", argIndex,
			))
		} else {
			conditions = append(conditions, fmt.Sprintf("o.category = $%d", argIndex))
		}
		args = append(args, filter.Category)
		argIndex++
	}

	if filter.OpportunityType != "" {
		conditions = append(conditions, fmt.Sprintf("o.opportunity_type = $%d", argIndex))
		args = append(args, filter.OpportunityType)
		argIndex++
	}

	if filter.ApplicationStatus != "" && filter.CatalogScope == CatalogScopeResearch {
		conditions = append(conditions, fmt.Sprintf(
			"COALESCE(o.type_metadata->>'application_status', 'unknown') = $%d", argIndex,
		))
		args = append(args, filter.ApplicationStatus)
		argIndex++
	}

	if filter.WorkArrangement != "" {
		if filter.CatalogScope == CatalogScopeAll {
			conditions = append(conditions, fmt.Sprintf(
				"(o.opportunity_type = 'research' OR o.work_arrangement = $%d)", argIndex,
			))
		} else {
			conditions = append(conditions, fmt.Sprintf("o.work_arrangement = $%d", argIndex))
		}
		args = append(args, filter.WorkArrangement)
		argIndex++
	}

	if filter.Location != "" {
		conditions = append(conditions, fmt.Sprintf("o.location ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Location+"%")
		argIndex++
	}

	if filter.CareerFamily != "" {
		conditions = append(conditions, fmt.Sprintf("o.career_family = $%d", argIndex))
		args = append(args, filter.CareerFamily)
		argIndex++
	}

	if filter.ExperienceLevel != "" {
		conditions = append(conditions, fmt.Sprintf("o.experience_level = $%d", argIndex))
		args = append(args, filter.ExperienceLevel)
		argIndex++
	}

	// Visibility: employment uses relevance tiers; research bypasses them.
	switch filter.CatalogScope {
	case CatalogScopeAll:
		conditions = append(conditions, fmt.Sprintf("(%s OR %s)",
			researchVisibilityCondition(),
			employmentVisibilityCondition(filter),
		))
	case CatalogScopeResearch:
		conditions = append(conditions, researchVisibilityCondition())
	default:
		conditions = append(conditions, employmentVisibilityCondition(filter))
	}

	return strings.Join(conditions, " AND "), args
}

func researchOrderBy() string {
	return `CASE COALESCE(o.type_metadata->>'application_status', 'unknown')
		WHEN 'open' THEN 1
		WHEN 'upcoming' THEN 2
		WHEN 'unknown' THEN 3
		WHEN 'closed' THEN 4
		ELSE 5
	END, o.deadline ASC NULLS LAST, o.created_at DESC`
}

func researchVisibilityCondition() string {
	return "o.opportunity_type = 'research'"
}

func employmentVisibilityCondition(filter ListFilter) string {
	parts := []string{"o.opportunity_type = 'employment'"}
	if filter.IncludeNonTechnical {
		if filter.IncludeAmbiguous {
			parts = append(parts, "o.relevance_tier IN ('high_confidence_technical', 'ambiguous', 'high_confidence_non_technical')")
		} else {
			parts = append(parts, "o.relevance_tier IN ('high_confidence_technical', 'high_confidence_non_technical')")
		}
	} else if filter.IncludeAmbiguous {
		parts = append(parts, "o.relevance_tier IN ('high_confidence_technical', 'ambiguous')")
	} else {
		parts = append(parts, "o.relevance_tier = 'high_confidence_technical'")
	}
	return strings.Join(parts, " AND ")
}

func normalizeSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
