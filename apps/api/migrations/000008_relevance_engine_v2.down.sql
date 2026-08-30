DROP INDEX IF EXISTS opportunities_experience_level_idx;
DROP INDEX IF EXISTS opportunities_career_family_idx;
DROP INDEX IF EXISTS opportunities_relevance_tier_idx;

ALTER TABLE opportunities DROP CONSTRAINT IF EXISTS opportunities_relevance_tier_check;
ALTER TABLE opportunities DROP CONSTRAINT IF EXISTS opportunities_education_level_check;
ALTER TABLE opportunities DROP CONSTRAINT IF EXISTS opportunities_career_family_check;
ALTER TABLE opportunities DROP CONSTRAINT IF EXISTS opportunities_experience_level_check;

ALTER TABLE opportunities
    DROP COLUMN IF EXISTS classification_reasons,
    DROP COLUMN IF EXISTS relevance_tier,
    DROP COLUMN IF EXISTS education_level,
    DROP COLUMN IF EXISTS career_family,
    DROP COLUMN IF EXISTS experience_level;
