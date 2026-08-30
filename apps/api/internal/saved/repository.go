package saved

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles saved opportunity persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new saved Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Save bookmarks an opportunity for a student (idempotent).
func (r *Repository) Save(ctx context.Context, studentID, opportunityID uuid.UUID) (*SavedOpportunity, error) {
	exists, err := r.opportunityExists(ctx, opportunityID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOpportunityNotFound
	}

	var saved SavedOpportunity
	err = r.pool.QueryRow(ctx, `
		INSERT INTO saved_opportunities (student_id, opportunity_id)
		VALUES ($1, $2)
		ON CONFLICT (student_id, opportunity_id) DO UPDATE
			SET saved_at = saved_opportunities.saved_at
		RETURNING id, opportunity_id, saved_at
	`, studentID, opportunityID).Scan(&saved.ID, &saved.OpportunityID, &saved.SavedAt)
	if err != nil {
		return nil, fmt.Errorf("save opportunity: %w", err)
	}
	return &saved, nil
}

// Unsave removes a bookmark for a student.
func (r *Repository) Unsave(ctx context.Context, studentID, opportunityID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM saved_opportunities
		WHERE student_id = $1 AND opportunity_id = $2
	`, studentID, opportunityID)
	if err != nil {
		return fmt.Errorf("unsave opportunity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaveNotFound
	}
	return nil
}

// List returns paginated saved opportunities for a student.
func (r *Repository) List(ctx context.Context, studentID uuid.UUID, filter ListFilter) ([]Summary, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM saved_opportunities so
		JOIN opportunities o ON o.id = so.opportunity_id
		WHERE so.student_id = $1
	`, studentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count saved: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.title, o.organization_name, o.category, o.opportunity_type,
		       o.location, o.work_arrangement, o.deadline, o.skills, o.tags, o.status,
		       o.verification_status, o.source, o.last_checked_at,
		       true AS is_saved,
		       EXISTS(
		           SELECT 1 FROM applications a
		           WHERE a.opportunity_id = o.id AND a.student_id = $1
		       ) AS has_application
		FROM saved_opportunities so
		JOIN opportunities o ON o.id = so.opportunity_id
		WHERE so.student_id = $1
		ORDER BY so.saved_at DESC
		LIMIT $2 OFFSET $3
	`, studentID, filter.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list saved: %w", err)
	}
	defer rows.Close()

	var results []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(
			&s.ID, &s.Title, &s.OrganizationName, &s.Category, &s.OpportunityType,
			&s.Location, &s.WorkArrangement, &s.Deadline, &s.Skills, &s.Tags, &s.Status,
			&s.VerificationStatus, &s.SourceName, &s.LastCheckedAt, &s.IsSaved, &s.HasApplication,
		); err != nil {
			return nil, 0, fmt.Errorf("scan saved: %w", err)
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

func (r *Repository) opportunityExists(ctx context.Context, opportunityID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM opportunities WHERE id = $1)
	`, opportunityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check opportunity: %w", err)
	}
	return exists, nil
}

func normalizeSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
