DELETE FROM employer_boards
WHERE opportunity_source_id IN (
    'c3000000-0000-4000-8000-000000000020',
    'c3000000-0000-4000-8000-000000000021',
    'c3000000-0000-4000-8000-000000000022',
    'c3000000-0000-4000-8000-000000000023',
    'c3000000-0000-4000-8000-000000000024',
    'c3000000-0000-4000-8000-000000000025'
);

DELETE FROM opportunity_sources
WHERE id IN (
    'c3000000-0000-4000-8000-000000000020',
    'c3000000-0000-4000-8000-000000000021',
    'c3000000-0000-4000-8000-000000000022',
    'c3000000-0000-4000-8000-000000000023',
    'c3000000-0000-4000-8000-000000000024',
    'c3000000-0000-4000-8000-000000000025'
);

ALTER TABLE employer_boards
    DROP CONSTRAINT employer_boards_ats_provider_check;

ALTER TABLE employer_boards
    ADD CONSTRAINT employer_boards_ats_provider_check
        CHECK (ats_provider IN ('greenhouse', 'lever'));

ALTER TABLE opportunity_sources
    DROP CONSTRAINT opportunity_sources_adapter_check;

ALTER TABLE opportunity_sources
    ADD CONSTRAINT opportunity_sources_adapter_check
        CHECK (adapter IN ('usajobs', 'greenhouse', 'lever', 'manual', 'dev_seed'));

ALTER TABLE ingestion_runs
    DROP COLUMN records_raw_fetched,
    DROP COLUMN records_retained,
    DROP COLUMN records_filtered_out;
