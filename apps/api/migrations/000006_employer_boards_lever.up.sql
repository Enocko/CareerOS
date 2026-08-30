-- Lever ATS provider + curated employer boards

INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES
    (
        'c3000000-0000-4000-8000-000000000030',
        'Lever · Palantir',
        'api',
        'lever',
        '{"board_token":"palantir","employer_name":"Palantir","source_url":"https://jobs.lever.co/palantir","tags":["technology","defense"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000031',
        'Lever · Shield AI',
        'api',
        'lever',
        '{"board_token":"shieldai","employer_name":"Shield AI","source_url":"https://jobs.lever.co/shieldai","tags":["technology","defense","ai"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000032',
        'Lever · Spotify',
        'api',
        'lever',
        '{"board_token":"spotify","employer_name":"Spotify","source_url":"https://jobs.lever.co/spotify","tags":["technology","media"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000033',
        'Lever · Gopuff',
        'api',
        'lever',
        '{"board_token":"gopuff","employer_name":"Gopuff","source_url":"https://jobs.lever.co/gopuff","tags":["technology","logistics"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000034',
        'Lever · Unlimit',
        'api',
        'lever',
        '{"board_token":"unlimit","employer_name":"Unlimit","source_url":"https://jobs.lever.co/unlimit","tags":["technology","fintech"]}'::jsonb,
        true,
        360
    );

INSERT INTO employer_boards (id, employer_name, ats_provider, board_token, source_url, tags, enabled, opportunity_source_id)
VALUES
    ('c4000000-0000-4000-8000-000000000030', 'Palantir',  'lever', 'palantir', 'https://jobs.lever.co/palantir', ARRAY['technology','defense'],        true, 'c3000000-0000-4000-8000-000000000030'),
    ('c4000000-0000-4000-8000-000000000031', 'Shield AI', 'lever', 'shieldai', 'https://jobs.lever.co/shieldai', ARRAY['technology','defense','ai'], true, 'c3000000-0000-4000-8000-000000000031'),
    ('c4000000-0000-4000-8000-000000000032', 'Spotify',   'lever', 'spotify',  'https://jobs.lever.co/spotify',  ARRAY['technology','media'],         true, 'c3000000-0000-4000-8000-000000000032'),
    ('c4000000-0000-4000-8000-000000000033', 'Gopuff',    'lever', 'gopuff',   'https://jobs.lever.co/gopuff',   ARRAY['technology','logistics'],     true, 'c3000000-0000-4000-8000-000000000033'),
    ('c4000000-0000-4000-8000-000000000034', 'Unlimit',   'lever', 'unlimit',  'https://jobs.lever.co/unlimit',  ARRAY['technology','fintech'],       true, 'c3000000-0000-4000-8000-000000000034');
