-- Relevance Engine v2: separate experience level, career family, education, and feed tier.

ALTER TABLE opportunities
    ADD COLUMN IF NOT EXISTS experience_level VARCHAR(50),
    ADD COLUMN IF NOT EXISTS career_family VARCHAR(80),
    ADD COLUMN IF NOT EXISTS education_level VARCHAR(50),
    ADD COLUMN IF NOT EXISTS relevance_tier VARCHAR(80),
    ADD COLUMN IF NOT EXISTS classification_reasons TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN (
            'internship', 'co_op', 'new_grad', 'early_career',
            'apprenticeship', 'fellowship', 'unknown'
        ));

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_career_family_check
        CHECK (career_family IS NULL OR career_family IN (
            'software_engineering', 'data_science', 'machine_learning_ai',
            'cybersecurity', 'product_management_technical',
            'cloud_infrastructure_devops', 'quantitative_technology',
            'technical_research', 'other_technical', 'non_technical', 'unknown'
        ));

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_education_level_check
        CHECK (education_level IS NULL OR education_level IN (
            'undergraduate', 'masters', 'phd', 'graduate_any', 'unspecified'
        ));

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_relevance_tier_check
        CHECK (relevance_tier IS NULL OR relevance_tier IN (
            'high_confidence_technical', 'ambiguous', 'high_confidence_non_technical'
        ));

CREATE INDEX IF NOT EXISTS opportunities_relevance_tier_idx ON opportunities (relevance_tier);
CREATE INDEX IF NOT EXISTS opportunities_career_family_idx ON opportunities (career_family);
CREATE INDEX IF NOT EXISTS opportunities_experience_level_idx ON opportunities (experience_level);

COMMENT ON COLUMN opportunities.experience_level IS 'Student/early-career experience band (Relevance Engine v2)';
COMMENT ON COLUMN opportunities.career_family IS 'Technical career family for personalization (Relevance Engine v2)';
COMMENT ON COLUMN opportunities.education_level IS 'Detected education requirement when present (Relevance Engine v2)';
COMMENT ON COLUMN opportunities.relevance_tier IS 'Product feed inclusion tier; source truth is independent of tier';
COMMENT ON COLUMN opportunities.classification_reasons IS 'Explainability reason codes from deterministic classifier';
