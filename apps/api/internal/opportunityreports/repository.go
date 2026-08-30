package opportunityreports

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists opportunity issue reports.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a reports repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new pending report.
func (r *Repository) Create(ctx context.Context, opportunityID, reporterID uuid.UUID, reason string, note *string) (*Report, error) {
	var rep Report
	err := r.pool.QueryRow(ctx, `
		INSERT INTO opportunity_reports (opportunity_id, reporter_id, reason, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, opportunity_id, reporter_id, reason, note, status, created_at, resolved_at, resolved_by
	`, opportunityID, reporterID, reason, note).Scan(
		&rep.ID, &rep.OpportunityID, &rep.ReporterID, &rep.Reason, &rep.Note,
		&rep.Status, &rep.CreatedAt, &rep.ResolvedAt, &rep.ResolvedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}
	return &rep, nil
}

// OpportunityExists checks that an opportunity row exists.
func (r *Repository) OpportunityExists(ctx context.Context, opportunityID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM opportunities WHERE id = $1)
	`, opportunityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check opportunity: %w", err)
	}
	return exists, nil
}

// CountPending returns unresolved report count for admin overview.
func (r *Repository) CountPending(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunity_reports WHERE status = $1
	`, StatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending reports: %w", err)
	}
	return count, nil
}

// ListQueue returns grouped pending reports for admin triage.
func (r *Repository) ListQueue(ctx context.Context, limit int) ([]QueueItem, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			o.id,
			o.title,
			o.organization_name,
			o.source,
			o.source_url,
			o.status,
			o.verification_status,
			o.last_checked_at,
			COUNT(r.id) AS report_count,
			MAX(r.created_at) AS latest_report_at,
			array_agg(DISTINCT r.reason ORDER BY r.reason) AS reasons
		FROM opportunity_reports r
		JOIN opportunities o ON o.id = r.opportunity_id
		WHERE r.status = $1
		GROUP BY o.id, o.title, o.organization_name, o.source, o.source_url,
		         o.status, o.verification_status, o.last_checked_at
		ORDER BY report_count DESC, latest_report_at DESC
		LIMIT $2
	`, StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("list report queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		if err := rows.Scan(
			&item.OpportunityID, &item.OpportunityTitle, &item.OrganizationName,
			&item.SourceName, &item.SourceURL, &item.OpportunityStatus,
			&item.VerificationStatus, &item.LastCheckedAt,
			&item.ReportCount, &item.LatestReportAt, &item.Reasons,
		); err != nil {
			return nil, fmt.Errorf("scan report queue: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, rows.Err()
}

// UpdateStatus resolves or dismisses all pending reports for an opportunity.
func (r *Repository) UpdateOpportunityReportsStatus(
	ctx context.Context,
	opportunityID uuid.UUID,
	status string,
	resolverID uuid.UUID,
) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE opportunity_reports
		SET status = $2,
		    resolved_at = now(),
		    resolved_by = $3
		WHERE opportunity_id = $1 AND status = $4
	`, opportunityID, status, resolverID, StatusPending)
	if err != nil {
		return 0, fmt.Errorf("update report status: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// GetByID returns a single report.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	var rep Report
	err := r.pool.QueryRow(ctx, `
		SELECT id, opportunity_id, reporter_id, reason, note, status, created_at, resolved_at, resolved_by
		FROM opportunity_reports
		WHERE id = $1
	`, id).Scan(
		&rep.ID, &rep.OpportunityID, &rep.ReporterID, &rep.Reason, &rep.Note,
		&rep.Status, &rep.CreatedAt, &rep.ResolvedAt, &rep.ResolvedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get report: %w", err)
	}
	return &rep, nil
}
