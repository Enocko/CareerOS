package catalogreport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository runs catalog health diagnostics.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a catalog report repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Metrics holds aggregate catalog health counts.
type Metrics struct {
	EmploymentVisible  int            `json:"employment_visible"`
	ResearchCandidate  int            `json:"research_candidate"`
	ResearchOpen       int            `json:"research_open"`
	ResearchUpcoming   int            `json:"research_upcoming"`
	ResearchUnknown    int            `json:"research_unknown"`
	ResearchClosed     int            `json:"research_closed"`
	VerifiedListings   int            `json:"verified_listings"`
	StaleListings      int            `json:"stale_listings"`
	ClosedListings     int            `json:"closed_listings"`
	WithDeadline       int            `json:"with_deadline"`
	WithoutDeadline    int            `json:"without_deadline"`
	CheckedWithin7Days int            `json:"checked_within_7_days"`
	PendingReports     int            `json:"pending_reports"`
	ProviderCounts     map[string]int `json:"provider_counts"`
}

// DuplicateGroup is a deterministic duplicate candidate cluster.
type DuplicateGroup struct {
	Organization string
	Title        string
	Count        int
	IDs          []uuid.UUID
}

// SourceHealthRow summarizes per-source ingestion health.
type SourceHealthRow struct {
	SourceName        string     `json:"source_name"`
	Adapter           string     `json:"adapter"`
	Enabled           bool       `json:"enabled"`
	LastSuccessAt     *time.Time `json:"last_success_at,omitempty"`
	LastFailedAt      *time.Time `json:"last_failed_at,omitempty"`
	LastRunStatus     *string    `json:"last_run_status,omitempty"`
	LastErrorCode     *string    `json:"last_error_code,omitempty"`
	LastErrorMessage  *string    `json:"last_error_message,omitempty"`
	VerifiedOpenCount int        `json:"verified_open_count"`
	ConsecutiveFails  int        `json:"consecutive_fails"`
}

// URLIssue flags suspicious application URLs.
type URLIssue struct {
	ID               uuid.UUID
	Title            string
	Source           string
	ApplicationURL   string
	Issue            string
}

const testIDFilter = `(
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
		o.external_id NOT LIKE 'LEVER-OK-BOARD-%'
	)
)`

// CollectMetrics aggregates catalog health counts.
func (r *Repository) CollectMetrics(ctx context.Context) (*Metrics, error) {
	m := &Metrics{ProviderCounts: map[string]int{}}
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE o.opportunity_type = 'employment' AND o.status = 'open' AND o.verification_status = 'verified' AND %s),
			COUNT(*) FILTER (WHERE o.opportunity_type = 'research' AND o.status = 'open' AND o.verification_status = 'verified' AND %s),
			COUNT(*) FILTER (WHERE o.opportunity_type = 'research' AND o.status = 'open' AND o.verification_status = 'verified'
				AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'open' AND %s),
			COUNT(*) FILTER (WHERE o.opportunity_type = 'research' AND o.status = 'open' AND o.verification_status = 'verified'
				AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'upcoming' AND %s),
			COUNT(*) FILTER (WHERE o.opportunity_type = 'research' AND o.status = 'open' AND o.verification_status = 'verified'
				AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'unknown' AND %s),
			COUNT(*) FILTER (WHERE o.opportunity_type = 'research' AND o.status = 'open' AND o.verification_status = 'verified'
				AND COALESCE(o.type_metadata->>'application_status', 'unknown') = 'closed' AND %s),
			COUNT(*) FILTER (WHERE o.verification_status = 'verified' AND o.status = 'open' AND %s),
			COUNT(*) FILTER (WHERE o.verification_status = 'stale' AND %s),
			COUNT(*) FILTER (WHERE o.status = 'closed' AND %s),
			COUNT(*) FILTER (WHERE o.deadline IS NOT NULL AND o.status = 'open' AND o.verification_status = 'verified' AND %s),
			COUNT(*) FILTER (WHERE o.deadline IS NULL AND o.status = 'open' AND o.verification_status = 'verified' AND %s),
			COUNT(*) FILTER (WHERE o.last_checked_at >= now() - interval '7 days' AND o.status = 'open' AND o.verification_status = 'verified' AND %s)
		FROM opportunities o
	`, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter, testIDFilter)).Scan(
		&m.EmploymentVisible, &m.ResearchCandidate, &m.ResearchOpen, &m.ResearchUpcoming,
		&m.ResearchUnknown, &m.ResearchClosed, &m.VerifiedListings, &m.StaleListings,
		&m.ClosedListings, &m.WithDeadline, &m.WithoutDeadline, &m.CheckedWithin7Days,
	)
	if err != nil {
		return nil, fmt.Errorf("collect metrics: %w", err)
	}

	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunity_reports WHERE status = 'pending'
	`).Scan(&m.PendingReports)

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT o.source, COUNT(*)
		FROM opportunities o
		WHERE o.status = 'open' AND o.verification_status = 'verified' AND o.opportunity_type = 'employment' AND %s
		GROUP BY o.source ORDER BY COUNT(*) DESC
	`, testIDFilter))
	if err != nil {
		return nil, fmt.Errorf("provider counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		m.ProviderCounts[name] = count
	}
	return m, rows.Err()
}

// FindDuplicateGroups returns deterministic employment duplicate clusters.
// The second int is total duplicate groups; the third is estimated excess records.
func (r *Repository) FindDuplicateGroups(ctx context.Context, limit int) ([]DuplicateGroup, int, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var totalGroups, totalRecords int
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		WITH employment AS (
			SELECT
				o.id,
				lower(trim(regexp_replace(o.organization_name, '[^a-zA-Z0-9]+', ' ', 'g'))) AS norm_org,
				lower(trim(regexp_replace(o.title, '[^a-zA-Z0-9]+', ' ', 'g'))) AS norm_title
			FROM opportunities o
			WHERE o.opportunity_type = 'employment'
			  AND o.status = 'open'
			  AND o.verification_status = 'verified'
			  AND %s
		),
		groups AS (
			SELECT norm_org, norm_title, COUNT(*) AS cnt, array_agg(id) AS ids
			FROM employment
			WHERE norm_org <> '' AND norm_title <> ''
			GROUP BY norm_org, norm_title
			HAVING COUNT(*) > 1
		)
		SELECT
			(SELECT COUNT(*) FROM groups),
			COALESCE((SELECT SUM(cnt - 1) FROM groups), 0)
	`, testIDFilter)).Scan(&totalGroups, &totalRecords)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count duplicate groups: %w", err)
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		WITH employment AS (
			SELECT
				o.id,
				o.organization_name,
				o.title,
				lower(trim(regexp_replace(o.organization_name, '[^a-zA-Z0-9]+', ' ', 'g'))) AS norm_org,
				lower(trim(regexp_replace(o.title, '[^a-zA-Z0-9]+', ' ', 'g'))) AS norm_title
			FROM opportunities o
			WHERE o.opportunity_type = 'employment'
			  AND o.status = 'open'
			  AND o.verification_status = 'verified'
			  AND %s
		)
		SELECT organization_name, title, COUNT(*) AS cnt, array_agg(id)
		FROM employment
		WHERE norm_org <> '' AND norm_title <> ''
		GROUP BY norm_org, norm_title, organization_name, title
		HAVING COUNT(*) > 1
		ORDER BY cnt DESC, organization_name, title
		LIMIT $1
	`, testIDFilter), limit)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list duplicate groups: %w", err)
	}
	defer rows.Close()

	var groups []DuplicateGroup
	for rows.Next() {
		var g DuplicateGroup
		if err := rows.Scan(&g.Organization, &g.Title, &g.Count, &g.IDs); err != nil {
			return nil, 0, 0, err
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []DuplicateGroup{}
	}
	return groups, totalGroups, totalRecords, rows.Err()
}

// ListSourceHealth returns per-source ingestion health.
func (r *Repository) ListSourceHealth(ctx context.Context) ([]SourceHealthRow, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			s.name,
			s.adapter,
			s.enabled,
			ls.finished_at,
			lf.finished_at,
			lr.status,
			lr.error_code,
			lr.error_message,
			COALESCE(vc.cnt, 0),
			COALESCE(cf.consecutive_fails, 0)
		FROM opportunity_sources s
		LEFT JOIN LATERAL (
			SELECT finished_at FROM ingestion_runs
			WHERE source_id = s.id AND status = 'success'
			ORDER BY finished_at DESC NULLS LAST LIMIT 1
		) ls ON true
		LEFT JOIN LATERAL (
			SELECT finished_at FROM ingestion_runs
			WHERE source_id = s.id AND status = 'failed'
			ORDER BY finished_at DESC NULLS LAST LIMIT 1
		) lf ON true
		LEFT JOIN LATERAL (
			SELECT status, error_code, error_message FROM ingestion_runs
			WHERE source_id = s.id AND finished_at IS NOT NULL
			ORDER BY finished_at DESC LIMIT 1
		) lr ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM opportunities o
			WHERE o.source_id = s.id AND o.status = 'open' AND o.verification_status = 'verified' AND %s
		) vc ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS consecutive_fails
			FROM (
				SELECT status FROM ingestion_runs
				WHERE source_id = s.id AND finished_at IS NOT NULL
				ORDER BY finished_at DESC LIMIT 5
			) recent
			WHERE status = 'failed'
		) cf ON true
		ORDER BY s.name
	`, testIDFilter))
	if err != nil {
		return nil, fmt.Errorf("list source health: %w", err)
	}
	defer rows.Close()

	var items []SourceHealthRow
	for rows.Next() {
		var row SourceHealthRow
		if err := rows.Scan(
			&row.SourceName, &row.Adapter, &row.Enabled,
			&row.LastSuccessAt, &row.LastFailedAt,
			&row.LastRunStatus, &row.LastErrorCode, &row.LastErrorMessage,
			&row.VerifiedOpenCount, &row.ConsecutiveFails,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if items == nil {
		items = []SourceHealthRow{}
	}
	return items, rows.Err()
}

// AuditApplicationURLs samples URLs and flags obvious issues.
func (r *Repository) AuditApplicationURLs(ctx context.Context, perProvider int) ([]URLIssue, error) {
	if perProvider <= 0 {
		perProvider = 10
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		WITH ranked AS (
			SELECT
				o.id, o.title, o.source, o.application_url,
				ROW_NUMBER() OVER (PARTITION BY o.source ORDER BY random()) AS rn
			FROM opportunities o
			WHERE o.status = 'open'
			  AND o.verification_status = 'verified'
			  AND o.application_url IS NOT NULL
			  AND o.application_url <> ''
			  AND %s
		)
		SELECT id, title, source, application_url
		FROM ranked
		WHERE rn <= $1
		ORDER BY source, title
	`, testIDFilter), perProvider)
	if err != nil {
		return nil, fmt.Errorf("audit urls: %w", err)
	}
	defer rows.Close()

	var issues []URLIssue
	for rows.Next() {
		var id uuid.UUID
		var title, source, url string
		if err := rows.Scan(&id, &title, &source, &url); err != nil {
			return nil, err
		}
		if issue := classifyURL(url); issue != "" {
			issues = append(issues, URLIssue{
				ID: id, Title: title, Source: source, ApplicationURL: url, Issue: issue,
			})
		}
	}
	if issues == nil {
		issues = []URLIssue{}
	}
	return issues, rows.Err()
}

func classifyURL(url string) string {
	lower := strings.ToLower(strings.TrimSpace(url))
	switch {
	case lower == "":
		return "empty url"
	case strings.Contains(lower, "localhost"), strings.Contains(lower, "127.0.0.1"):
		return "localhost/test host"
	case strings.Contains(lower, "etap.nsf.gov") && !strings.Contains(lower, "/reu/"):
		return "generic ETAP homepage"
	case strings.Contains(lower, "boards.greenhouse.io") && !strings.Contains(lower, "/jobs/"):
		return "generic Greenhouse board"
	case strings.Contains(lower, "jobs.ashbyhq.com") && strings.Count(lower, "/") < 4:
		return "possibly generic Ashby board"
	case !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://"):
		return "missing http(s) scheme"
	}
	return ""
}
