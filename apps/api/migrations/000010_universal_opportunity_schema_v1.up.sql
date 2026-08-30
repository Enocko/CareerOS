-- Universal Opportunity Schema v1 (Model C) — additive migration

-- ---------------------------------------------------------------------------
-- 1. Add nullable columns
-- ---------------------------------------------------------------------------
ALTER TABLE opportunities
    ADD COLUMN IF NOT EXISTS opportunity_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS type_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS verification_method VARCHAR(50),
    ADD COLUMN IF NOT EXISTS employment_mode VARCHAR(50);

-- ---------------------------------------------------------------------------
-- 2. Backfill opportunity_type from legacy category (deterministic)
-- ---------------------------------------------------------------------------
UPDATE opportunities
SET opportunity_type = CASE category
    WHEN 'fellowship' THEN 'fellowship'
    WHEN 'scholarship' THEN 'scholarship'
    WHEN 'research' THEN 'research'
    WHEN 'hackathon' THEN 'event'
    WHEN 'conference' THEN 'event'
    WHEN 'leadership_program' THEN 'program'
    WHEN 'internship' THEN 'employment'
    WHEN 'full_time' THEN 'employment'
    WHEN 'part_time' THEN 'employment'
    WHEN 'apprenticeship' THEN 'employment'
    ELSE 'employment'
END
WHERE opportunity_type IS NULL;

-- ---------------------------------------------------------------------------
-- 3. Backfill employment_mode for employment rows
-- ---------------------------------------------------------------------------
UPDATE opportunities
SET employment_mode = CASE category
    WHEN 'full_time' THEN 'full_time'
    WHEN 'part_time' THEN 'part_time'
    ELSE NULL
END
WHERE opportunity_type = 'employment'
  AND employment_mode IS NULL;

-- ---------------------------------------------------------------------------
-- 4. Clear invalid employment experience_level value before constraint change
-- ---------------------------------------------------------------------------
UPDATE opportunities
SET experience_level = NULL
WHERE experience_level = 'fellowship';

-- ---------------------------------------------------------------------------
-- 5. Backfill verification_method
-- ---------------------------------------------------------------------------
UPDATE opportunities
SET verification_method = CASE
    WHEN verification_status = 'verified' AND source_id IS NOT NULL THEN 'official_source'
    WHEN verification_status = 'verified' AND source_id IS NULL THEN 'manual_verified'
    ELSE NULL
END
WHERE verification_method IS NULL;

-- ---------------------------------------------------------------------------
-- 6. Enforce NOT NULL on opportunity_type after backfill
-- ---------------------------------------------------------------------------
ALTER TABLE opportunities
    ALTER COLUMN opportunity_type SET NOT NULL;

ALTER TABLE opportunities
    ALTER COLUMN opportunity_type SET DEFAULT 'employment';

-- ---------------------------------------------------------------------------
-- 7. Constraints
-- ---------------------------------------------------------------------------
ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_opportunity_type_check
        CHECK (opportunity_type IN (
            'employment', 'research', 'scholarship', 'fellowship',
            'program', 'event', 'competition', 'other'
        ));

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_employment_mode_check
        CHECK (employment_mode IS NULL OR employment_mode IN (
            'full_time', 'part_time', 'seasonal'
        ));

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_verification_method_check
        CHECK (verification_method IS NULL OR verification_method IN (
            'official_source', 'partner', 'manual_verified', 'community_verified'
        ));

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_experience_level_check;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN (
            'internship', 'co_op', 'new_grad', 'early_career',
            'apprenticeship', 'unknown'
        ));

CREATE INDEX IF NOT EXISTS opportunities_opportunity_type_idx
    ON opportunities (opportunity_type);

COMMENT ON COLUMN opportunities.opportunity_type IS 'Canonical opportunity kind (Model C). Employment seniority lives in experience_level.';
COMMENT ON COLUMN opportunities.type_metadata IS 'Type-specific JSON validated in application layer.';
COMMENT ON COLUMN opportunities.verification_method IS 'How CareerOS established provenance (complements verification_status).';
COMMENT ON COLUMN opportunities.employment_mode IS 'Employment schedule mode; only meaningful when opportunity_type = employment.';
COMMENT ON COLUMN opportunities.category IS 'DEPRECATED: legacy compatibility alias; do not use for new logic.';
