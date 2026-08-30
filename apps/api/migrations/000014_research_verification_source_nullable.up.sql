-- Allow NULL verification_source_url when availability cannot be established (unknown status).

ALTER TABLE research_availability_verifications
    ALTER COLUMN verification_source_url DROP NOT NULL;
