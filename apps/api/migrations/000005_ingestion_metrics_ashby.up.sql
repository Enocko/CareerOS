-- Ingestion observability metrics + Ashby ATS provider

ALTER TABLE ingestion_runs
    ADD COLUMN records_raw_fetched INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN records_retained INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN records_filtered_out INTEGER NOT NULL DEFAULT 0;

-- Backfill retained metrics from legacy records_fetched column.
UPDATE ingestion_runs
SET records_retained = records_fetched,
    records_raw_fetched = records_fetched
WHERE records_retained = 0 AND records_fetched > 0;

ALTER TABLE opportunity_sources
    DROP CONSTRAINT opportunity_sources_adapter_check;

ALTER TABLE opportunity_sources
    ADD CONSTRAINT opportunity_sources_adapter_check
        CHECK (adapter IN ('usajobs', 'greenhouse', 'lever', 'ashby', 'manual', 'dev_seed'));

ALTER TABLE employer_boards
    DROP CONSTRAINT employer_boards_ats_provider_check;

ALTER TABLE employer_boards
    ADD CONSTRAINT employer_boards_ats_provider_check
        CHECK (ats_provider IN ('greenhouse', 'lever', 'ashby'));

-- Each Ashby board is a separate opportunity_source for per-board sync isolation.
INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES
    (
        'c3000000-0000-4000-8000-000000000020',
        'Ashby · Notion',
        'api',
        'ashby',
        '{"board_token":"notion","employer_name":"Notion","source_url":"https://jobs.ashbyhq.com/notion","tags":["technology","product"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000021',
        'Ashby · Ramp',
        'api',
        'ashby',
        '{"board_token":"ramp","employer_name":"Ramp","source_url":"https://jobs.ashbyhq.com/ramp","tags":["technology","fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000022',
        'Ashby · OpenAI',
        'api',
        'ashby',
        '{"board_token":"openai","employer_name":"OpenAI","source_url":"https://jobs.ashbyhq.com/openai","tags":["technology","ai"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000023',
        'Ashby · Plaid',
        'api',
        'ashby',
        '{"board_token":"plaid","employer_name":"Plaid","source_url":"https://jobs.ashbyhq.com/plaid","tags":["technology","fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000024',
        'Ashby · Linear',
        'api',
        'ashby',
        '{"board_token":"linear","employer_name":"Linear","source_url":"https://jobs.ashbyhq.com/linear","tags":["technology","product"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000025',
        'Ashby · Ashby',
        'api',
        'ashby',
        '{"board_token":"ashby","employer_name":"Ashby","source_url":"https://jobs.ashbyhq.com/ashby","tags":["technology"]}'::jsonb,
        true,
        360
    );

INSERT INTO employer_boards (id, employer_name, ats_provider, board_token, source_url, tags, enabled, opportunity_source_id)
VALUES
    ('c4000000-0000-4000-8000-000000000020', 'Notion', 'ashby', 'notion', 'https://jobs.ashbyhq.com/notion', ARRAY['technology','product'], true, 'c3000000-0000-4000-8000-000000000020'),
    ('c4000000-0000-4000-8000-000000000021', 'Ramp',   'ashby', 'ramp',   'https://jobs.ashbyhq.com/ramp',   ARRAY['technology','fintech'], true, 'c3000000-0000-4000-8000-000000000021'),
    ('c4000000-0000-4000-8000-000000000022', 'OpenAI', 'ashby', 'openai', 'https://jobs.ashbyhq.com/openai', ARRAY['technology','ai'],      true, 'c3000000-0000-4000-8000-000000000022'),
    ('c4000000-0000-4000-8000-000000000023', 'Plaid',  'ashby', 'plaid',  'https://jobs.ashbyhq.com/plaid',  ARRAY['technology','fintech'], true, 'c3000000-0000-4000-8000-000000000023'),
    ('c4000000-0000-4000-8000-000000000024', 'Linear', 'ashby', 'linear', 'https://jobs.ashbyhq.com/linear', ARRAY['technology','product'], true, 'c3000000-0000-4000-8000-000000000024'),
    ('c4000000-0000-4000-8000-000000000025', 'Ashby',  'ashby', 'ashby',  'https://jobs.ashbyhq.com/ashby',  ARRAY['technology'],           true, 'c3000000-0000-4000-8000-000000000025');
