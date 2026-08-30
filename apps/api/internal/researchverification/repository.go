package researchverification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/careeros/api/internal/opportunitytype"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists research verification workflow data.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a research verification repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListQueue returns prioritized NSF REU candidates for verification.
func (r *Repository) ListQueue(ctx context.Context, limit int) ([]QueueItem, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.title, o.organization_name, o.source_url,
		       o.type_metadata->>'program_url' AS program_url,
		       COALESCE(o.type_metadata->>'application_status', 'unknown') AS application_status,
		       o.type_metadata,
		       (
		         CASE WHEN o.type_metadata->>'next_verification_at' IS NOT NULL
		              AND (o.type_metadata->>'next_verification_at')::timestamptz <= now() THEN 40 ELSE 0 END
		         + CASE WHEN o.type_metadata->>'program_url' IS NOT NULL THEN 30 ELSE 0 END
		         + CASE WHEN o.type_metadata->>'application_verified_at' IS NULL THEN 20 ELSE 0 END
		         + CASE WHEN COALESCE(o.type_metadata->>'application_status', 'unknown') = 'unknown' THEN 10 ELSE 0 END
		       ) AS priority_score,
		       NULLIF(o.type_metadata->>'application_verified_at', '')::timestamptz AS last_verified_at,
		       NULLIF(o.type_metadata->>'next_verification_at', '')::timestamptz AS next_verification_at
		FROM opportunities o
		JOIN opportunity_sources os ON os.id = o.source_id AND os.adapter = 'nsf_reu'
		WHERE o.verification_status = 'verified'
		  AND o.status = 'open'
		  AND o.opportunity_type = 'research'
		ORDER BY priority_score DESC, o.last_seen_at DESC NULLS LAST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list verification queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		var meta []byte
		if err := rows.Scan(
			&item.ID, &item.Title, &item.OrganizationName, &item.SourceURL,
			&item.ProgramURL, &item.ApplicationStatus, &meta,
			&item.PriorityScore, &item.LastVerifiedAt, &item.NextVerificationAt,
		); err != nil {
			return nil, fmt.Errorf("scan queue item: %w", err)
		}
		if len(meta) > 0 {
			item.TypeMetadata = meta
		}
		items = append(items, item)
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, rows.Err()
}

// GetResearchOpportunity returns a research opportunity if it exists.
func (r *Repository) GetResearchOpportunity(ctx context.Context, id uuid.UUID) (*QueueItem, error) {
	var item QueueItem
	var meta []byte
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.title, o.organization_name, o.source_url,
		       o.type_metadata->>'program_url',
		       COALESCE(o.type_metadata->>'application_status', 'unknown'),
		       o.type_metadata,
		       0,
		       NULLIF(o.type_metadata->>'application_verified_at', '')::timestamptz,
		       NULLIF(o.type_metadata->>'next_verification_at', '')::timestamptz
		FROM opportunities o
		WHERE o.id = $1 AND o.opportunity_type = 'research'
	`, id).Scan(
		&item.ID, &item.Title, &item.OrganizationName, &item.SourceURL,
		&item.ProgramURL, &item.ApplicationStatus, &meta,
		&item.PriorityScore, &item.LastVerifiedAt, &item.NextVerificationAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("research opportunity not found")
		}
		return nil, fmt.Errorf("get research opportunity: %w", err)
	}
	if len(meta) > 0 {
		item.TypeMetadata = meta
	}
	return &item, nil
}

// ListVerifications returns verification history for an opportunity.
func (r *Repository) ListVerifications(ctx context.Context, opportunityID uuid.UUID, limit int) ([]VerificationRecord, error) {
	if limit < 1 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, opportunity_id, application_status, application_url, verification_source_url,
		       opens_at, deadline, cycle_label, verification_method, verified_at, verified_by,
		       next_verification_at, notes
		FROM research_availability_verifications
		WHERE opportunity_id = $1
		ORDER BY verified_at DESC
		LIMIT $2
	`, opportunityID, limit)
	if err != nil {
		return nil, fmt.Errorf("list verifications: %w", err)
	}
	defer rows.Close()

	var records []VerificationRecord
	for rows.Next() {
		var rec VerificationRecord
		if err := rows.Scan(
			&rec.ID, &rec.OpportunityID, &rec.ApplicationStatus, &rec.ApplicationURL,
			&rec.VerificationSourceURL, &rec.OpensAt, &rec.Deadline, &rec.CycleLabel,
			&rec.VerificationMethod, &rec.VerifiedAt, &rec.VerifiedBy,
			&rec.NextVerificationAt, &rec.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan verification: %w", err)
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []VerificationRecord{}
	}
	return records, rows.Err()
}

type applyVerificationInput struct {
	OpportunityID         uuid.UUID
	ApplicationStatus     string
	ApplicationURL        *string
	VerificationSourceURL *string
	OpensAt               *time.Time
	Deadline              *time.Time
	CycleLabel            *string
	VerificationMethod    string
	VerifiedBy            uuid.UUID
	NextVerificationAt    time.Time
	Notes                 *string
	TypeMetadata          json.RawMessage
	OpportunityDeadline   *time.Time
	Now                   time.Time
}

// ApplyVerification inserts audit record and updates opportunity current cycle state.
func (r *Repository) ApplyVerification(ctx context.Context, in applyVerificationInput) (*VerificationRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var rec VerificationRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO research_availability_verifications (
			opportunity_id, application_status, application_url, verification_source_url,
			opens_at, deadline, cycle_label, verification_method, verified_by,
			next_verification_at, notes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, opportunity_id, application_status, application_url, verification_source_url,
		          opens_at, deadline, cycle_label, verification_method, verified_at, verified_by,
		          next_verification_at, notes
	`, in.OpportunityID, in.ApplicationStatus, in.ApplicationURL, in.VerificationSourceURL,
		in.OpensAt, in.Deadline, in.CycleLabel, in.VerificationMethod, in.VerifiedBy,
		in.NextVerificationAt, in.Notes,
	).Scan(
		&rec.ID, &rec.OpportunityID, &rec.ApplicationStatus, &rec.ApplicationURL,
		&rec.VerificationSourceURL, &rec.OpensAt, &rec.Deadline, &rec.CycleLabel,
		&rec.VerificationMethod, &rec.VerifiedAt, &rec.VerifiedBy,
		&rec.NextVerificationAt, &rec.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("insert verification: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE opportunities
		SET type_metadata = $2,
		    application_url = $3,
		    deadline = $4,
		    last_checked_at = $5,
		    updated_at = $5
		WHERE id = $1 AND opportunity_type = 'research'
	`, in.OpportunityID, in.TypeMetadata, in.ApplicationURL, in.OpportunityDeadline, in.Now)
	if err != nil {
		return nil, fmt.Errorf("update opportunity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit verification: %w", err)
	}
	return &rec, nil
}

// GetMetrics returns research verification catalog metrics.
func (r *Repository) GetMetrics(ctx context.Context) (Metrics, error) {
	var m Metrics
	err := r.pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) AS candidates,
		  COUNT(*) FILTER (WHERE COALESCE(o.type_metadata->>'application_status','unknown') = 'unknown') AS unknown_count,
		  COUNT(*) FILTER (WHERE o.type_metadata->>'application_verified_at' IS NOT NULL) AS verified_programs,
		  COUNT(*) FILTER (WHERE o.type_metadata->>'application_status' = 'open') AS open_count,
		  COUNT(*) FILTER (WHERE o.type_metadata->>'application_status' = 'upcoming') AS upcoming_count,
		  COUNT(*) FILTER (WHERE o.type_metadata->>'application_status' = 'closed') AS closed_count,
		  COUNT(*) FILTER (WHERE o.application_url IS NOT NULL) AS app_urls,
		  COUNT(*) FILTER (WHERE o.deadline IS NOT NULL AND o.type_metadata->>'application_status' IN ('open','upcoming')) AS deadlines,
		  COUNT(*) FILTER (
		    WHERE o.type_metadata->>'next_verification_at' IS NOT NULL
		      AND (o.type_metadata->>'next_verification_at')::timestamptz <= now()
		      AND o.type_metadata->>'application_verified_at' IS NOT NULL
		  ) AS stale_count
		FROM opportunities o
		JOIN opportunity_sources os ON os.id = o.source_id AND os.adapter = 'nsf_reu'
		WHERE o.verification_status = 'verified' AND o.status = 'open'
	`).Scan(
		&m.CandidatePrograms, &m.AvailabilityUnknown, &m.VerifiedPrograms,
		&m.ApplicationsOpen, &m.ApplicationsUpcoming, &m.ApplicationsClosed,
		&m.DirectApplicationURLs, &m.VerifiedDeadlines, &m.VerificationStale,
	)
	if err != nil {
		return m, fmt.Errorf("get metrics: %w", err)
	}
	return m, nil
}

// BuildTypeMetadata merges current cycle fields into research type_metadata JSON.
func BuildTypeMetadata(existing json.RawMessage, status, method, sourceURL, cycleLabel, opensAt, verifiedAt, nextVerification string, programURL *string) (json.RawMessage, error) {
	meta := map[string]any{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &meta)
	}
	meta["application_status"] = status
	meta["availability_verification_method"] = method
	meta["application_status_method"] = method
	if sourceURL != "" {
		meta["application_verification_source_url"] = sourceURL
	} else {
		delete(meta, "application_verification_source_url")
	}
	if cycleLabel != "" {
		meta["cycle_label"] = cycleLabel
	} else {
		delete(meta, "cycle_label")
	}
	if opensAt != "" {
		meta["opens_at"] = opensAt
	} else {
		delete(meta, "opens_at")
	}
	if verifiedAt != "" {
		meta["application_verified_at"] = verifiedAt
	}
	if nextVerification != "" {
		meta["next_verification_at"] = nextVerification
	}
	if programURL != nil && *programURL != "" {
		meta["program_url"] = *programURL
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	if err := opportunitytype.ValidateWrite(opportunitytype.WriteInput{
		OpportunityType: opportunitytype.Research,
		TypeMetadata:    raw,
	}); err != nil {
		return nil, err
	}
	return raw, nil
}
