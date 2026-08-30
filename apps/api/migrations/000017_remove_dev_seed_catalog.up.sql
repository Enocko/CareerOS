-- Remove development seed catalog from production databases.
-- Safe to run in all environments: deletes only legacy manual/dev_seed fixtures.

DELETE FROM saved_opportunities
WHERE opportunity_id IN (
    SELECT id FROM opportunities
    WHERE source IN ('manual', 'dev_seed')
       OR organization_id IN (
           'a1000000-0000-4000-8000-000000000001',
           'a1000000-0000-4000-8000-000000000002',
           'a1000000-0000-4000-8000-000000000003',
           'a1000000-0000-4000-8000-000000000004',
           'a1000000-0000-4000-8000-000000000005',
           'a1000000-0000-4000-8000-000000000006',
           'a1000000-0000-4000-8000-000000000007'
       )
);

DELETE FROM applications
WHERE opportunity_id IN (
    SELECT id FROM opportunities
    WHERE source IN ('manual', 'dev_seed')
       OR organization_id IN (
           'a1000000-0000-4000-8000-000000000001',
           'a1000000-0000-4000-8000-000000000002',
           'a1000000-0000-4000-8000-000000000003',
           'a1000000-0000-4000-8000-000000000004',
           'a1000000-0000-4000-8000-000000000005',
           'a1000000-0000-4000-8000-000000000006',
           'a1000000-0000-4000-8000-000000000007'
       )
);

DELETE FROM opportunity_views
WHERE opportunity_id IN (
    SELECT id FROM opportunities
    WHERE source IN ('manual', 'dev_seed')
       OR organization_id IN (
           'a1000000-0000-4000-8000-000000000001',
           'a1000000-0000-4000-8000-000000000002',
           'a1000000-0000-4000-8000-000000000003',
           'a1000000-0000-4000-8000-000000000004',
           'a1000000-0000-4000-8000-000000000005',
           'a1000000-0000-4000-8000-000000000006',
           'a1000000-0000-4000-8000-000000000007'
       )
);

DELETE FROM opportunities
WHERE source IN ('manual', 'dev_seed')
   OR organization_id IN (
       'a1000000-0000-4000-8000-000000000001',
       'a1000000-0000-4000-8000-000000000002',
       'a1000000-0000-4000-8000-000000000003',
       'a1000000-0000-4000-8000-000000000004',
       'a1000000-0000-4000-8000-000000000005',
       'a1000000-0000-4000-8000-000000000006',
       'a1000000-0000-4000-8000-000000000007'
   );

DELETE FROM organizations
WHERE id IN (
    'a1000000-0000-4000-8000-000000000001',
    'a1000000-0000-4000-8000-000000000002',
    'a1000000-0000-4000-8000-000000000003',
    'a1000000-0000-4000-8000-000000000004',
    'a1000000-0000-4000-8000-000000000005',
    'a1000000-0000-4000-8000-000000000006',
    'a1000000-0000-4000-8000-000000000007'
);

DELETE FROM opportunity_sources WHERE adapter = 'dev_seed';
