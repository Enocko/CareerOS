UPDATE research_availability_verifications
SET verification_source_url = ''
WHERE verification_source_url IS NULL;

ALTER TABLE research_availability_verifications
    ALTER COLUMN verification_source_url SET NOT NULL;
