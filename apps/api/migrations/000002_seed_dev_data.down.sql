-- Remove development seed data only.
DELETE FROM opportunities WHERE source = 'manual' AND id::text LIKE 'b2000000%';
DELETE FROM organizations WHERE id::text LIKE 'a1000000%';
