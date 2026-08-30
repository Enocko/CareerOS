DELETE FROM opportunity_sources WHERE adapter = 'nsf_reu';

ALTER TABLE opportunity_sources
    DROP CONSTRAINT opportunity_sources_adapter_check;

ALTER TABLE opportunity_sources
    ADD CONSTRAINT opportunity_sources_adapter_check
        CHECK (adapter IN ('usajobs', 'greenhouse', 'lever', 'ashby', 'manual', 'dev_seed'));
