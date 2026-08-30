-- Research Discovery v1: NSF REU Site ingestion source

ALTER TABLE opportunity_sources
    DROP CONSTRAINT opportunity_sources_adapter_check;

ALTER TABLE opportunity_sources
    ADD CONSTRAINT opportunity_sources_adapter_check
        CHECK (adapter IN ('usajobs', 'greenhouse', 'lever', 'ashby', 'manual', 'dev_seed', 'nsf_reu'));

INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES (
    'c3000000-0000-4000-8000-000000000002',
    'U.S. National Science Foundation',
    'api',
    'nsf_reu',
    '{
        "keyword": "\"REU Site\"",
        "fund_program_name": "RSCH EXPER FOR UNDERGRAD SITES",
        "results_per_page": 25,
        "base_url": "https://api.nsf.gov/services/v1/awards.json"
    }'::jsonb,
    true,
    1440
);
