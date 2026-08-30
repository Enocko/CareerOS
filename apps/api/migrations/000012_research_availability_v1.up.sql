-- Research Availability Verification v1: correct NSF REU application semantics

UPDATE opportunities o
SET
    application_url = NULL,
    type_metadata = COALESCE(o.type_metadata, '{}'::jsonb)
        || jsonb_build_object(
            'application_status', 'unknown',
            'application_status_method', 'nsf_award_only',
            'availability_verification_method', 'unknown'
        )
        || CASE
            WHEN o.application_url IS NOT NULL
                 AND o.application_url NOT ILIKE '%etap.nsf.gov%'
                 AND o.application_url NOT ILIKE '%awardsearch%'
                 AND o.application_url NOT ILIKE '%nsf.gov/award%'
            THEN jsonb_build_object('program_url', o.application_url)
            ELSE '{}'::jsonb
        END
FROM opportunity_sources os
WHERE o.source_id = os.id
  AND os.adapter = 'nsf_reu'
  AND o.opportunity_type = 'research';
