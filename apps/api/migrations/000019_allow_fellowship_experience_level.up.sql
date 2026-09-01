-- Align DB constraint with application-layer experience levels (fixes Palantir/Scale/DoorDash ingest).
ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_experience_level_check;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_experience_level_check
        CHECK (experience_level IS NULL OR experience_level IN (
            'internship', 'co_op', 'new_grad', 'early_career',
            'apprenticeship', 'fellowship', 'unknown'
        ));
