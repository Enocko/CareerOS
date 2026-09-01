UPDATE opportunities SET experience_level = 'early_career' WHERE experience_level = 'fellowship';

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_experience_level_check;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN (
            'internship', 'co_op', 'new_grad', 'early_career',
            'apprenticeship', 'unknown'
        ));
