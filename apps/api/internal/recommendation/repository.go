package recommendation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository loads recommendation candidates from PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a recommendation repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListCandidates returns verified technical opportunities for scoring.
func (r *Repository) ListCandidates(ctx context.Context, studentID uuid.UUID) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.title, o.organization_name, o.category, o.location,
		       o.work_arrangement, o.deadline, o.skills, o.tags, o.status,
		       o.verification_status, o.source, o.last_checked_at,
		       o.experience_level, o.career_family, o.education_level, o.relevance_tier,
		       (so.id IS NOT NULL) AS is_saved,
		       EXISTS(
		           SELECT 1 FROM applications a
		           WHERE a.opportunity_id = o.id AND a.student_id = $1
		       ) AS has_application
		FROM opportunities o
		LEFT JOIN saved_opportunities so
			ON so.opportunity_id = o.id AND so.student_id = $1
		WHERE o.status = 'open'
		  AND o.verification_status = 'verified'
		  AND o.opportunity_type = 'employment'
		  AND o.relevance_tier = 'high_confidence_technical'
		  AND (
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
		  )
	`, studentID)
	if err != nil {
		return nil, fmt.Errorf("list recommendation candidates: %w", err)
	}
	defer rows.Close()

	var items []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(
			&c.ID, &c.Title, &c.OrganizationName, &c.Category, &c.Location,
			&c.WorkArrangement, &c.Deadline, &c.Skills, &c.Tags, &c.Status,
			&c.VerificationStatus, &c.SourceName, &c.LastCheckedAt,
			&c.ExperienceLevel, &c.CareerFamily, &c.EducationLevel, &c.RelevanceTier,
			&c.IsSaved, &c.HasApplication,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation candidate: %w", err)
		}
		if c.Skills == nil {
			c.Skills = []string{}
		}
		if c.Tags == nil {
			c.Tags = []string{}
		}
		items = append(items, c)
	}
	if items == nil {
		items = []Candidate{}
	}
	return items, rows.Err()
}

// CountCandidates returns the number of eligible recommendation candidates.
func (r *Repository) CountCandidates(ctx context.Context, studentID uuid.UUID, now time.Time) (int, error) {
	candidates, err := r.ListCandidates(ctx, studentID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range candidates {
		if PassesHardFilters(c, now) {
			count++
		}
	}
	return count, nil
}
