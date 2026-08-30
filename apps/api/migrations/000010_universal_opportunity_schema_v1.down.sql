DROP INDEX IF EXISTS opportunities_opportunity_type_idx;

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_verification_method_check;

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_employment_mode_check;

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_opportunity_type_check;

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_experience_level_check;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN (
            'internship', 'co_op', 'new_grad', 'early_career',
            'apprenticeship', 'fellowship', 'unknown'
        ));

ALTER TABLE opportunities
    DROP COLUMN IF EXISTS employment_mode,
    DROP COLUMN IF EXISTS verification_method,
    DROP COLUMN IF EXISTS type_metadata,
    DROP COLUMN IF EXISTS opportunity_type;
