-- Milestone 3: ATS ingestion — Greenhouse employer board registry

CREATE TABLE employer_boards (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employer_name         VARCHAR(255) NOT NULL,
    ats_provider          VARCHAR(50)  NOT NULL DEFAULT 'greenhouse',
    board_token           VARCHAR(255) NOT NULL,
    source_url            VARCHAR(1000) NOT NULL,
    tags                  TEXT[]       NOT NULL DEFAULT '{}',
    enabled               BOOLEAN      NOT NULL DEFAULT true,
    opportunity_source_id UUID         NOT NULL UNIQUE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT employer_boards_ats_provider_check
        CHECK (ats_provider IN ('greenhouse', 'lever')),
    CONSTRAINT employer_boards_opportunity_source_id_fkey
        FOREIGN KEY (opportunity_source_id) REFERENCES opportunity_sources(id) ON DELETE CASCADE,
    CONSTRAINT employer_boards_token_uniq
        UNIQUE (ats_provider, board_token)
);

CREATE INDEX employer_boards_enabled_idx ON employer_boards (enabled);

-- Each employer board is a separate opportunity_source for per-board sync isolation.
INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES
    (
        'c3000000-0000-4000-8000-000000000010',
        'Greenhouse · Stripe',
        'api',
        'greenhouse',
        '{"board_token":"stripe","employer_name":"Stripe","source_url":"https://boards.greenhouse.io/stripe","tags":["technology","fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000011',
        'Greenhouse · Datadog',
        'api',
        'greenhouse',
        '{"board_token":"datadog","employer_name":"Datadog","source_url":"https://boards.greenhouse.io/datadog","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000012',
        'Greenhouse · Cloudflare',
        'api',
        'greenhouse',
        '{"board_token":"cloudflare","employer_name":"Cloudflare","source_url":"https://boards.greenhouse.io/cloudflare","tags":["technology","cybersecurity"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000013',
        'Greenhouse · Figma',
        'api',
        'greenhouse',
        '{"board_token":"figma","employer_name":"Figma","source_url":"https://boards.greenhouse.io/figma","tags":["technology","product"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000014',
        'Greenhouse · Discord',
        'api',
        'greenhouse',
        '{"board_token":"discord","employer_name":"Discord","source_url":"https://boards.greenhouse.io/discord","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000015',
        'Greenhouse · Roblox',
        'api',
        'greenhouse',
        '{"board_token":"roblox","employer_name":"Roblox","source_url":"https://boards.greenhouse.io/roblox","tags":["technology","gaming"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000016',
        'Greenhouse · Coinbase',
        'api',
        'greenhouse',
        '{"board_token":"coinbase","employer_name":"Coinbase","source_url":"https://boards.greenhouse.io/coinbase","tags":["technology","finance"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000017',
        'Greenhouse · Dropbox',
        'api',
        'greenhouse',
        '{"board_token":"dropbox","employer_name":"Dropbox","source_url":"https://boards.greenhouse.io/dropbox","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000018',
        'Greenhouse · Block',
        'api',
        'greenhouse',
        '{"board_token":"block","employer_name":"Block","source_url":"https://boards.greenhouse.io/block","tags":["technology","fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000019',
        'Greenhouse · Lyft',
        'api',
        'greenhouse',
        '{"board_token":"lyft","employer_name":"Lyft","source_url":"https://boards.greenhouse.io/lyft","tags":["technology"]}'::jsonb,
        true,
        360
    );

INSERT INTO employer_boards (id, employer_name, ats_provider, board_token, source_url, tags, enabled, opportunity_source_id)
VALUES
    ('c4000000-0000-4000-8000-000000000010', 'Stripe',     'greenhouse', 'stripe',     'https://boards.greenhouse.io/stripe',     ARRAY['technology','fintech'],       true, 'c3000000-0000-4000-8000-000000000010'),
    ('c4000000-0000-4000-8000-000000000011', 'Datadog',    'greenhouse', 'datadog',    'https://boards.greenhouse.io/datadog',    ARRAY['technology'],                 true, 'c3000000-0000-4000-8000-000000000011'),
    ('c4000000-0000-4000-8000-000000000012', 'Cloudflare', 'greenhouse', 'cloudflare', 'https://boards.greenhouse.io/cloudflare', ARRAY['technology','cybersecurity'], true, 'c3000000-0000-4000-8000-000000000012'),
    ('c4000000-0000-4000-8000-000000000013', 'Figma',      'greenhouse', 'figma',      'https://boards.greenhouse.io/figma',      ARRAY['technology','product'],       true, 'c3000000-0000-4000-8000-000000000013'),
    ('c4000000-0000-4000-8000-000000000014', 'Discord',    'greenhouse', 'discord',    'https://boards.greenhouse.io/discord',    ARRAY['technology'],                 true, 'c3000000-0000-4000-8000-000000000014'),
    ('c4000000-0000-4000-8000-000000000015', 'Roblox',     'greenhouse', 'roblox',     'https://boards.greenhouse.io/roblox',     ARRAY['technology','gaming'],        true, 'c3000000-0000-4000-8000-000000000015'),
    ('c4000000-0000-4000-8000-000000000016', 'Coinbase',   'greenhouse', 'coinbase',   'https://boards.greenhouse.io/coinbase',   ARRAY['technology','finance'],       true, 'c3000000-0000-4000-8000-000000000016'),
    ('c4000000-0000-4000-8000-000000000017', 'Dropbox',    'greenhouse', 'dropbox',    'https://boards.greenhouse.io/dropbox',    ARRAY['technology'],                 true, 'c3000000-0000-4000-8000-000000000017'),
    ('c4000000-0000-4000-8000-000000000018', 'Block',      'greenhouse', 'block',      'https://boards.greenhouse.io/block',      ARRAY['technology','fintech'],       true, 'c3000000-0000-4000-8000-000000000018'),
    ('c4000000-0000-4000-8000-000000000019', 'Lyft',       'greenhouse', 'lyft',       'https://boards.greenhouse.io/lyft',       ARRAY['technology'],                 true, 'c3000000-0000-4000-8000-000000000019');
