package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/opportunitytype"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles ingestion persistence.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new ingestion Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListEnabledSources returns all enabled ingestion sources.
func (r *Repository) ListEnabledSources(ctx context.Context) ([]Source, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, source_type, adapter, config, enabled,
		       sync_interval_minutes, created_at, updated_at
		FROM opportunity_sources
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled sources: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SourceType, &s.Adapter, &s.Config, &s.Enabled,
			&s.SyncIntervalMinutes, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, s)
	}
	if sources == nil {
		sources = []Source{}
	}
	return sources, rows.Err()
}

// GetSourceByID returns a source by ID.
func (r *Repository) GetSourceByID(ctx context.Context, id uuid.UUID) (*Source, error) {
	var s Source
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, source_type, adapter, config, enabled,
		       sync_interval_minutes, created_at, updated_at
		FROM opportunity_sources
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.Name, &s.SourceType, &s.Adapter, &s.Config, &s.Enabled,
		&s.SyncIntervalMinutes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("source not found")
		}
		return nil, fmt.Errorf("get source: %w", err)
	}
	return &s, nil
}

// CreateRun inserts a new ingestion run in running state.
func (r *Repository) CreateRun(ctx context.Context, sourceID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ingestion_runs (source_id, status)
		VALUES ($1, $2)
		RETURNING id
	`, sourceID, RunStatusRunning).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create ingestion run: %w", err)
	}
	return id, nil
}

// FinishCounts holds per-run ingestion metrics persisted to ingestion_runs.
type FinishCounts struct {
	RawFetched  int
	Retained    int
	FilteredOut int
	Created     int
	Updated     int
	Stale       int
	Closed      int
}

// FinishRun updates an ingestion run with final status and counts.
func (r *Repository) FinishRun(
	ctx context.Context,
	runID uuid.UUID,
	status string,
	counts FinishCounts,
	errMsg, errCode *string,
) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ingestion_runs
		SET finished_at = now(),
		    status = $2,
		    records_raw_fetched = $3,
		    records_retained = $4,
		    records_filtered_out = $5,
		    records_fetched = $4,
		    records_created = $6,
		    records_updated = $7,
		    records_stale = $8,
		    records_closed = $9,
		    error_message = $10,
		    error_code = $11
		WHERE id = $1
	`, runID, status, counts.RawFetched, counts.Retained, counts.FilteredOut,
		counts.Created, counts.Updated, counts.Stale, counts.Closed, errMsg, errCode)
	if err != nil {
		return fmt.Errorf("finish ingestion run: %w", err)
	}
	return nil
}

// UpsertOpportunity inserts or updates an opportunity from ingestion.
func (r *Repository) UpsertOpportunity(
	ctx context.Context,
	sourceID uuid.UUID,
	sourceName string,
	raw ingestrecord.RawOpportunity,
	now time.Time,
) (created bool, err error) {
	var location, compensation, eligibility, applicationURL *string
	if raw.Location != "" {
		location = &raw.Location
	}
	if raw.Compensation != "" {
		compensation = &raw.Compensation
	}
	if raw.Eligibility != "" {
		eligibility = &raw.Eligibility
	}
	if raw.ApplicationURL != "" {
		applicationURL = &raw.ApplicationURL
	}

	oppType, employmentMode, verificationMethod := opportunitytype.IngestionDefaults(raw.Category)
	if raw.OpportunityType != "" {
		oppType = raw.OpportunityType
		employmentMode = ""
	}
	if raw.VerificationMethod != "" {
		verificationMethod = raw.VerificationMethod
	}
	typeMetadata := json.RawMessage(`{}`)
	if len(raw.TypeMetadata) > 0 {
		typeMetadata = raw.TypeMetadata
	}
	if err := opportunitytype.ValidateWrite(opportunitytype.WriteInput{
		OpportunityType:    oppType,
		ExperienceLevel:    raw.ExperienceLevel,
		EmploymentMode:     employmentMode,
		CareerFamily:       raw.CareerFamily,
		WorkArrangement:    raw.WorkArrangement,
		TypeMetadata:       typeMetadata,
		VerificationMethod: verificationMethod,
	}); err != nil {
		return false, fmt.Errorf("validate opportunity write: %w", err)
	}

	var id uuid.UUID
	err = r.pool.QueryRow(ctx, `
		INSERT INTO opportunities (
			source_id, external_id, title, organization_name, description,
			category, opportunity_type, type_metadata, verification_method, employment_mode,
			location, work_arrangement, deadline, skills, compensation,
			application_url, source_url, source, status, tags,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			missed_sync_count, experience_level, career_family, education_level,
			relevance_tier, classification_reasons, eligibility,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, 'open', $19,
			$20, $21, $22, $23,
			0, $24, $25, $26, $27, $28, $29,
			$21, $21
		)
		ON CONFLICT (source_id, external_id) WHERE source_id IS NOT NULL AND external_id IS NOT NULL
		DO UPDATE SET
			title = EXCLUDED.title,
			organization_name = EXCLUDED.organization_name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			opportunity_type = EXCLUDED.opportunity_type,
			type_metadata = EXCLUDED.type_metadata,
			verification_method = EXCLUDED.verification_method,
			employment_mode = EXCLUDED.employment_mode,
			location = EXCLUDED.location,
			work_arrangement = EXCLUDED.work_arrangement,
			deadline = EXCLUDED.deadline,
			skills = EXCLUDED.skills,
			compensation = EXCLUDED.compensation,
			application_url = EXCLUDED.application_url,
			source_url = EXCLUDED.source_url,
			source = EXCLUDED.source,
			tags = EXCLUDED.tags,
			status = 'open',
			verification_status = $20,
			experience_level = EXCLUDED.experience_level,
			career_family = EXCLUDED.career_family,
			education_level = EXCLUDED.education_level,
			relevance_tier = EXCLUDED.relevance_tier,
			classification_reasons = EXCLUDED.classification_reasons,
			eligibility = EXCLUDED.eligibility,
			last_seen_at = $22,
			last_checked_at = $23,
			missed_sync_count = 0,
			updated_at = $21
		RETURNING (xmax = 0) AS inserted, id
	`,
		sourceID, raw.ExternalID, raw.Title, raw.Organization, raw.Description,
		raw.Category, oppType, typeMetadata, verificationMethod, nullableString(employmentMode),
		location, raw.WorkArrangement, raw.Deadline, raw.Skills, compensation,
		applicationURL, raw.SourceURL, sourceName, raw.Tags,
		VerificationVerified, now, now, now,
		nullableString(raw.ExperienceLevel), nullableString(raw.CareerFamily),
		nullableString(raw.EducationLevel), nullableString(raw.RelevanceTier),
		raw.ClassificationReasons, eligibility,
	).Scan(&created, &id)
	if err != nil {
		return false, fmt.Errorf("upsert opportunity: %w", err)
	}
	return created, nil
}

// ApplyPostSyncActions marks missing opportunities stale and closes expired deadlines.
// Must only be called after a fully successful source fetch.
func (r *Repository) ApplyPostSyncActions(
	ctx context.Context,
	sourceID uuid.UUID,
	seenExternalIDs []string,
	now time.Time,
) (staleCount, closedCount int, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin post-sync tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE opportunities
		SET missed_sync_count = missed_sync_count + 1,
		    last_checked_at = $3,
		    updated_at = $3
		WHERE source_id = $1
		  AND verification_status = $2
		  AND status = 'open'
		  AND external_id IS NOT NULL
		  AND NOT (external_id = ANY($4))
	`, sourceID, VerificationVerified, now, seenExternalIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("increment missed sync count: %w", err)
	}
	_ = tag

	staleResult, err := tx.Exec(ctx, `
		UPDATE opportunities
		SET verification_status = $4,
		    updated_at = $3
		WHERE source_id = $1
		  AND verification_status = $2
		  AND status = 'open'
		  AND missed_sync_count >= $5
	`, sourceID, VerificationVerified, now, VerificationStale, StaleAfterMissedSyncs)
	if err != nil {
		return 0, 0, fmt.Errorf("mark stale: %w", err)
	}
	staleCount = int(staleResult.RowsAffected())

	closedResult, err := tx.Exec(ctx, `
		UPDATE opportunities
		SET verification_status = $4,
		    status = 'closed',
		    updated_at = $3
		WHERE source_id = $1
		  AND verification_status IN ($2, $5)
		  AND status = 'open'
		  AND deadline IS NOT NULL
		  AND deadline < ($3::timestamptz)::date
		  AND NOT (external_id = ANY($6))
	`, sourceID, VerificationVerified, now, VerificationClosed, VerificationStale, seenExternalIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("close expired deadlines: %w", err)
	}
	closedCount = int(closedResult.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit post-sync tx: %w", err)
	}
	return staleCount, closedCount, nil
}

// CountVerifiedBySource returns the number of verified open opportunities for a source.
func (r *Repository) CountVerifiedBySource(ctx context.Context, sourceID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM opportunities
		WHERE source_id = $1
		  AND verification_status = $2
		  AND status = 'open'
	`, sourceID, VerificationVerified).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count verified opportunities: %w", err)
	}
	return count, nil
}

// GetOpportunityVerification returns verification fields for test assertions.
func (r *Repository) GetOpportunityVerification(ctx context.Context, sourceID uuid.UUID, externalID string) (status string, missed int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT verification_status, missed_sync_count
		FROM opportunities
		WHERE source_id = $1 AND external_id = $2
	`, sourceID, externalID).Scan(&status, &missed)
	if err != nil {
		return "", 0, fmt.Errorf("get opportunity verification: %w", err)
	}
	return status, missed, nil
}

// GetOpportunityLifecycle returns catalog lifecycle fields for test assertions.
func (r *Repository) GetOpportunityLifecycle(
	ctx context.Context,
	sourceID uuid.UUID,
	externalID string,
) (id uuid.UUID, status, verificationStatus string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT id, status, verification_status
		FROM opportunities
		WHERE source_id = $1 AND external_id = $2
	`, sourceID, externalID).Scan(&id, &status, &verificationStatus)
	if err != nil {
		return uuid.Nil, "", "", fmt.Errorf("get opportunity lifecycle: %w", err)
	}
	return id, status, verificationStatus, nil
}

// InsertTestOpportunity inserts a verified opportunity for integration tests.
func (r *Repository) InsertTestOpportunity(ctx context.Context, sourceID uuid.UUID, externalID, title string) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO opportunities (
			source_id, external_id, title, organization_name, description,
			category, work_arrangement, application_url, source_url, source,
			verification_status, first_seen_at, last_seen_at, last_checked_at,
			status, skills, tags, missed_sync_count
		) VALUES (
			$1, $2, $3, 'Test Agency', 'Test description for ingestion.',
			'internship', 'remote', 'https://www.usajobs.gov/test', 'https://www.usajobs.gov/test',
			'USAJobs', $4, $5, $5, $5, 'open', '{}', '{}', 0
		)
		ON CONFLICT (source_id, external_id) WHERE source_id IS NOT NULL AND external_id IS NOT NULL
		DO UPDATE SET verification_status = EXCLUDED.verification_status,
		              status = 'open',
		              missed_sync_count = 0
	`, sourceID, externalID, title, VerificationVerified, now)
	if err != nil {
		return fmt.Errorf("insert test opportunity: %w", err)
	}
	return nil
}

// InsertTestSource registers an isolated ingestion source for automated tests.
func (r *Repository) InsertTestSource(ctx context.Context, id uuid.UUID) error {
	return r.InsertTestSourceWithAdapter(ctx, id, "usajobs")
}

// InsertTestSourceWithAdapter registers an isolated ingestion source with a specific adapter.
func (r *Repository) InsertTestSourceWithAdapter(ctx context.Context, id uuid.UUID, adapter string) error {
	return r.InsertTestSourceWithConfig(ctx, id, adapter, []byte(`{}`))
}

// InsertTestSourceWithConfig registers an isolated ingestion source with adapter config.
func (r *Repository) InsertTestSourceWithConfig(ctx context.Context, id uuid.UUID, adapter string, config []byte) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
		VALUES ($1, 'Automated Test Source', 'api', $2, $3, true, 360)
		ON CONFLICT (id) DO NOTHING
	`, id, adapter, config)
	if err != nil {
		return fmt.Errorf("insert test source: %w", err)
	}
	return nil
}

// DeleteTestSource removes a test source and its related data.
func (r *Repository) DeleteTestSource(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM ingestion_runs WHERE source_id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete test ingestion runs: %w", err)
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM opportunities WHERE source_id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete test opportunities: %w", err)
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM opportunity_sources WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete test source: %w", err)
	}
	return nil
}

// DeleteOpportunityByExternalID removes a single opportunity by source and external ID.
func (r *Repository) DeleteOpportunityByExternalID(ctx context.Context, sourceID uuid.UUID, externalID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM opportunities
		WHERE source_id = $1 AND external_id = $2
	`, sourceID, externalID)
	if err != nil {
		return fmt.Errorf("delete opportunity: %w", err)
	}
	return nil
}

// DeleteAllTestOpportunities removes opportunities created by automated tests.
func (r *Repository) DeleteAllTestOpportunities(ctx context.Context) (int64, error) {
	conditions := make([]string, 0, len(TestExternalIDPrefixes))
	args := make([]any, 0, len(TestExternalIDPrefixes))
	for i, prefix := range TestExternalIDPrefixes {
		conditions = append(conditions, fmt.Sprintf("external_id LIKE $%d", i+1))
		args = append(args, prefix+"%")
	}
	query := fmt.Sprintf(`DELETE FROM opportunities WHERE %s`, strings.Join(conditions, " OR "))
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete test opportunities: %w", err)
	}
	return tag.RowsAffected(), nil
}

// GetPreviousFinishedRun returns the most recent completed run before the current one.
func (r *Repository) GetPreviousFinishedRun(ctx context.Context, sourceID, currentRunID uuid.UUID) (*Run, error) {
	var run Run
	err := r.pool.QueryRow(ctx, `
		SELECT id, source_id, started_at, finished_at, status,
		       records_raw_fetched, records_retained, records_filtered_out,
		       records_fetched, records_created, records_updated,
		       records_stale, records_closed, error_message, error_code
		FROM ingestion_runs
		WHERE source_id = $1
		  AND id <> $2
		  AND status IN ($3, $4)
		ORDER BY started_at DESC
		LIMIT 1
	`, sourceID, currentRunID, RunStatusSuccess, RunStatusFailed).Scan(
		&run.ID, &run.SourceID, &run.StartedAt, &run.FinishedAt, &run.Status,
		&run.RecordsRawFetched, &run.RecordsRetained, &run.RecordsFilteredOut,
		&run.RecordsFetched, &run.RecordsCreated, &run.RecordsUpdated,
		&run.RecordsStale, &run.RecordsClosed, &run.ErrorMessage, &run.ErrorCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get previous finished run: %w", err)
	}
	return &run, nil
}

// GetLatestRun returns the most recent run for a source.
func (r *Repository) GetLatestRun(ctx context.Context, sourceID uuid.UUID) (*Run, error) {
	var run Run
	err := r.pool.QueryRow(ctx, `
		SELECT id, source_id, started_at, finished_at, status,
		       records_raw_fetched, records_retained, records_filtered_out,
		       records_fetched, records_created, records_updated,
		       records_stale, records_closed, error_message, error_code
		FROM ingestion_runs
		WHERE source_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, sourceID).Scan(
		&run.ID, &run.SourceID, &run.StartedAt, &run.FinishedAt, &run.Status,
		&run.RecordsRawFetched, &run.RecordsRetained, &run.RecordsFilteredOut,
		&run.RecordsFetched, &run.RecordsCreated, &run.RecordsUpdated,
		&run.RecordsStale, &run.RecordsClosed, &run.ErrorMessage, &run.ErrorCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest run: %w", err)
	}
	return &run, nil
}

// SourceCatalogCount holds verified opportunity counts for reporting.
type SourceCatalogCount struct {
	SourceID   uuid.UUID
	SourceName string
	Adapter    string
	Count      int
}

// CountVerifiedCatalog returns verified open opportunity counts grouped by source.
func (r *Repository) CountVerifiedCatalog(ctx context.Context) ([]SourceCatalogCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.name, s.adapter, COUNT(o.id)
		FROM opportunity_sources s
		LEFT JOIN opportunities o
			ON o.source_id = s.id
			AND o.verification_status = $1
			AND o.status = 'open'
		WHERE s.enabled = true
		GROUP BY s.id, s.name, s.adapter
		ORDER BY s.name
	`, VerificationVerified)
	if err != nil {
		return nil, fmt.Errorf("count verified catalog: %w", err)
	}
	defer rows.Close()

	var counts []SourceCatalogCount
	for rows.Next() {
		var item SourceCatalogCount
		if err := rows.Scan(&item.SourceID, &item.SourceName, &item.Adapter, &item.Count); err != nil {
			return nil, fmt.Errorf("scan verified catalog count: %w", err)
		}
		counts = append(counts, item)
	}
	if counts == nil {
		counts = []SourceCatalogCount{}
	}
	return counts, rows.Err()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// SampleRow is a title sample for manual audit reporting.
type SampleRow struct {
	Title         string
	CareerFamily  *string
	RelevanceTier *string
}

// SampleByRelevanceTier returns random opportunities for a relevance tier.
func (r *Repository) SampleByRelevanceTier(ctx context.Context, tier string, limit int) ([]SampleRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT title, career_family, relevance_tier
		FROM opportunities
		WHERE verification_status = $1
		  AND status = 'open'
		  AND relevance_tier = $2
		ORDER BY random()
		LIMIT $3
	`, VerificationVerified, tier, limit)
	if err != nil {
		return nil, fmt.Errorf("sample by tier: %w", err)
	}
	defer rows.Close()

	var samples []SampleRow
	for rows.Next() {
		var row SampleRow
		if err := rows.Scan(&row.Title, &row.CareerFamily, &row.RelevanceTier); err != nil {
			return nil, err
		}
		samples = append(samples, row)
	}
	if samples == nil {
		samples = []SampleRow{}
	}
	return samples, rows.Err()
}

type OpportunityRow struct {
	ID          uuid.UUID
	Title       string
	Description string
}

// ListVerifiedOpportunities returns verified open opportunities for reclassification.
func (r *Repository) ListVerifiedOpportunities(ctx context.Context) ([]OpportunityRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, description
		FROM opportunities
		WHERE verification_status = $1
		  AND status = 'open'
		ORDER BY title
	`, VerificationVerified)
	if err != nil {
		return nil, fmt.Errorf("list verified opportunities: %w", err)
	}
	defer rows.Close()

	var items []OpportunityRow
	for rows.Next() {
		var row OpportunityRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Description); err != nil {
			return nil, fmt.Errorf("scan opportunity row: %w", err)
		}
		items = append(items, row)
	}
	if items == nil {
		items = []OpportunityRow{}
	}
	return items, rows.Err()
}

// UpdateClassification persists v2 classification fields for an opportunity.
func (r *Repository) UpdateClassification(
	ctx context.Context,
	id uuid.UUID,
	experienceLevel, careerFamily, educationLevel, relevanceTier string,
	reasons []string,
) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE opportunities
		SET experience_level = $2,
		    career_family = $3,
		    education_level = $4,
		    relevance_tier = $5,
		    classification_reasons = $6,
		    updated_at = now()
		WHERE id = $1
	`, id, nullableString(experienceLevel), nullableString(careerFamily),
		nullableString(educationLevel), nullableString(relevanceTier), reasons)
	if err != nil {
		return fmt.Errorf("update classification: %w", err)
	}
	return nil
}

// ClassificationStats holds aggregate counts for reporting.
type ClassificationStats struct {
	Total                    int
	TechnicalFeed            int
	NonTechnical             int
	Ambiguous                int
	ByCareerFamily           map[string]int
	ByExperienceLevel        map[string]int
	ByEducationLevel         map[string]int
}

// CountClassificationStats aggregates v2 fields for verified open opportunities.
func (r *Repository) CountClassificationStats(ctx context.Context) (*ClassificationStats, error) {
	stats := &ClassificationStats{
		ByCareerFamily:    map[string]int{},
		ByExperienceLevel: map[string]int{},
		ByEducationLevel:  map[string]int{},
	}

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM opportunities
		WHERE verification_status = $1 AND status = 'open'
	`, VerificationVerified).Scan(&stats.Total)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT relevance_tier, COUNT(*)
		FROM opportunities
		WHERE verification_status = $1 AND status = 'open'
		GROUP BY relevance_tier
	`, VerificationVerified)
	if err != nil {
		return nil, fmt.Errorf("count tiers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tier *string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			return nil, err
		}
		key := "unset"
		if tier != nil {
			key = *tier
		}
		switch key {
		case "high_confidence_technical":
			stats.TechnicalFeed = count
		case "high_confidence_non_technical":
			stats.NonTechnical = count
		case "ambiguous":
			stats.Ambiguous = count
		}
	}

	for _, query := range []struct {
		column string
		dest   map[string]int
	}{
		{"career_family", stats.ByCareerFamily},
		{"experience_level", stats.ByExperienceLevel},
		{"education_level", stats.ByEducationLevel},
	} {
		qrows, qerr := r.pool.Query(ctx, fmt.Sprintf(`
			SELECT %s, COUNT(*)
			FROM opportunities
			WHERE verification_status = $1 AND status = 'open'
			GROUP BY %s
		`, query.column, query.column), VerificationVerified)
		if qerr != nil {
			return nil, qerr
		}
		for qrows.Next() {
			var key *string
			var count int
			if err := qrows.Scan(&key, &count); err != nil {
				qrows.Close()
				return nil, err
			}
			label := "unset"
			if key != nil && *key != "" {
				label = *key
			}
			query.dest[label] = count
		}
		qrows.Close()
		if err := qrows.Err(); err != nil {
			return nil, err
		}
	}

	return stats, nil
}

// ListDueSources returns enabled sources past their sync interval without a running sync.
func (r *Repository) ListDueSources(ctx context.Context, now time.Time) ([]Source, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.name, s.source_type, s.adapter, s.config, s.enabled,
		       s.sync_interval_minutes, s.created_at, s.updated_at
		FROM opportunity_sources s
		LEFT JOIN LATERAL (
			SELECT started_at, finished_at, status
			FROM ingestion_runs
			WHERE source_id = s.id
			ORDER BY started_at DESC
			LIMIT 1
		) lr ON true
		WHERE s.enabled = true
		  AND NOT (lr.status = $1 AND lr.finished_at IS NULL)
		  AND (
			lr.finished_at IS NULL OR
			lr.finished_at < $2 - (s.sync_interval_minutes || ' minutes')::interval
		  )
		ORDER BY s.name
	`, RunStatusRunning, now)
	if err != nil {
		return nil, fmt.Errorf("list due sources: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SourceType, &s.Adapter, &s.Config, &s.Enabled,
			&s.SyncIntervalMinutes, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	if sources == nil {
		sources = []Source{}
	}
	return sources, rows.Err()
}

// HasRunningIngestion reports whether a source currently has a running ingestion run.
func (r *Repository) HasRunningIngestion(ctx context.Context, sourceID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ingestion_runs
			WHERE source_id = $1 AND status = $2 AND finished_at IS NULL
		)
	`, sourceID, RunStatusRunning).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has running ingestion: %w", err)
	}
	return exists, nil
}

// LastSuccessfulIngestionBySource returns last successful run time per enabled source.
func (r *Repository) LastSuccessfulIngestionBySource(ctx context.Context) (map[string]time.Time, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.name, MAX(r.finished_at)
		FROM opportunity_sources s
		LEFT JOIN ingestion_runs r ON r.source_id = s.id AND r.status = $1
		WHERE s.enabled = true
		GROUP BY s.name
		ORDER BY s.name
	`, RunStatusSuccess)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var name string
		var finished *time.Time
		if err := rows.Scan(&name, &finished); err != nil {
			return nil, err
		}
		if finished != nil {
			out[name] = *finished
		}
	}
	return out, rows.Err()
}

// SourceHealthStats summarizes latest ingestion outcomes for enabled sources.
type SourceHealthStats struct {
	SuccessSources int
	FailedSources  int
}

// IngestionSourceHealth counts enabled sources by latest run outcome.
func (r *Repository) IngestionSourceHealth(ctx context.Context) (SourceHealthStats, error) {
	var stats SourceHealthStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE lr.status = $1),
			COUNT(*) FILTER (WHERE lr.status = $2)
		FROM opportunity_sources s
		LEFT JOIN LATERAL (
			SELECT status
			FROM ingestion_runs
			WHERE source_id = s.id AND finished_at IS NOT NULL
			ORDER BY finished_at DESC
			LIMIT 1
		) lr ON true
		WHERE s.enabled = true
	`, RunStatusSuccess, RunStatusFailed).Scan(&stats.SuccessSources, &stats.FailedSources)
	if err != nil {
		return stats, fmt.Errorf("ingestion source health: %w", err)
	}
	return stats, nil
}
