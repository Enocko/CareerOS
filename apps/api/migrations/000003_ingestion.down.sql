DROP INDEX IF EXISTS opportunities_last_seen_at_idx;
DROP INDEX IF EXISTS opportunities_verification_status_idx;
DROP INDEX IF EXISTS opportunities_source_external_id_uniq;

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_verification_status_check,
    DROP CONSTRAINT IF EXISTS opportunities_source_id_fkey;

ALTER TABLE opportunities
    DROP COLUMN IF EXISTS missed_sync_count,
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS first_seen_at,
    DROP COLUMN IF EXISTS verification_status,
    DROP COLUMN IF EXISTS source_url,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS source_id;

ALTER TABLE opportunities
    RENAME COLUMN last_checked_at TO last_verified;

UPDATE opportunities
SET source = 'manual'
WHERE source = 'dev_seed';

DROP TABLE IF EXISTS ingestion_runs;
DROP TABLE IF EXISTS opportunity_sources;
