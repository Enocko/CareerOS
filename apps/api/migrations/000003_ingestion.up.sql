-- Milestone 2A/2B: Opportunity ingestion foundation

-- ---------------------------------------------------------------------------
-- opportunity_sources
-- ---------------------------------------------------------------------------
CREATE TABLE opportunity_sources (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  VARCHAR(255) NOT NULL,
    source_type           VARCHAR(50)  NOT NULL DEFAULT 'api',
    adapter               VARCHAR(100) NOT NULL,
    config                JSONB        NOT NULL DEFAULT '{}',
    enabled               BOOLEAN      NOT NULL DEFAULT true,
    sync_interval_minutes INTEGER      NOT NULL DEFAULT 360,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT opportunity_sources_source_type_check
        CHECK (source_type IN ('api', 'manual', 'organization_submission')),
    CONSTRAINT opportunity_sources_adapter_check
        CHECK (adapter IN ('usajobs', 'greenhouse', 'lever', 'manual', 'dev_seed'))
);

-- ---------------------------------------------------------------------------
-- ingestion_runs
-- ---------------------------------------------------------------------------
CREATE TABLE ingestion_runs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id        UUID NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at      TIMESTAMPTZ,
    status           VARCHAR(20) NOT NULL DEFAULT 'running',
    records_fetched  INTEGER NOT NULL DEFAULT 0,
    records_created  INTEGER NOT NULL DEFAULT 0,
    records_updated  INTEGER NOT NULL DEFAULT 0,
    records_stale    INTEGER NOT NULL DEFAULT 0,
    records_closed   INTEGER NOT NULL DEFAULT 0,
    error_message    TEXT,
    error_code       VARCHAR(100),
    CONSTRAINT ingestion_runs_source_id_fkey
        FOREIGN KEY (source_id) REFERENCES opportunity_sources(id) ON DELETE CASCADE,
    CONSTRAINT ingestion_runs_status_check
        CHECK (status IN ('running', 'success', 'failed'))
);

CREATE INDEX ingestion_runs_source_id_idx ON ingestion_runs (source_id);
CREATE INDEX ingestion_runs_started_at_idx ON ingestion_runs (started_at DESC);

-- ---------------------------------------------------------------------------
-- Extend opportunities
-- ---------------------------------------------------------------------------
ALTER TABLE opportunities
    RENAME COLUMN last_verified TO last_checked_at;

ALTER TABLE opportunities
    ADD COLUMN source_id UUID,
    ADD COLUMN external_id VARCHAR(500),
    ADD COLUMN source_url VARCHAR(1000),
    ADD COLUMN verification_status VARCHAR(20) NOT NULL DEFAULT 'unverified',
    ADD COLUMN first_seen_at TIMESTAMPTZ,
    ADD COLUMN last_seen_at TIMESTAMPTZ,
    ADD COLUMN missed_sync_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_source_id_fkey
        FOREIGN KEY (source_id) REFERENCES opportunity_sources(id) ON DELETE SET NULL,
    ADD CONSTRAINT opportunities_verification_status_check
        CHECK (verification_status IN ('verified', 'unverified', 'stale', 'closed'));

CREATE UNIQUE INDEX opportunities_source_external_id_uniq
    ON opportunities (source_id, external_id)
    WHERE source_id IS NOT NULL AND external_id IS NOT NULL;

CREATE INDEX opportunities_verification_status_idx
    ON opportunities (verification_status, status);

CREATE INDEX opportunities_last_seen_at_idx ON opportunities (last_seen_at);

-- Mark development seed data as unverified and hide from default browse.
UPDATE opportunities
SET
    source = 'dev_seed',
    verification_status = 'unverified',
    last_checked_at = NULL,
    last_seen_at = NULL
WHERE source = 'manual';

-- Register USAJobs as the first ingestion source.
INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES (
    'c3000000-0000-4000-8000-000000000001',
    'USAJobs',
    'api',
    'usajobs',
    '{"keyword": "intern", "hiring_path": "student", "results_per_page": 100}'::jsonb,
    true,
    360
);
