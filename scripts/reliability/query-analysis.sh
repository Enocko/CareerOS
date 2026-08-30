#!/usr/bin/env bash
set -euo pipefail

run_sql() {
  docker exec -i careeros-postgres psql -U careeros -d careeros -v ON_ERROR_STOP=1 "$@"
}

run_sql <<'SQL'
\timing on
\echo '=== opportunities browse count ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT COUNT(*)
FROM opportunities o
WHERE o.status = 'open'
  AND o.verification_status = 'verified'
  AND (
    o.external_id IS NULL OR (
      o.external_id NOT LIKE 'API-TEST-%' AND
      o.external_id NOT LIKE 'UPSERT-TEST-%'
    )
  );

\echo '=== opportunities browse list ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT o.id, o.title, o.organization_name, o.category, o.location,
       o.work_arrangement, o.deadline, o.skills, o.tags, o.status,
       o.verification_status, o.source, o.last_checked_at,
       o.experience_level, o.career_family, o.relevance_tier,
       false AS is_saved
FROM opportunities o
WHERE o.status = 'open'
  AND o.verification_status = 'verified'
ORDER BY o.deadline ASC NULLS LAST, o.created_at DESC
LIMIT 20 OFFSET 0;

\echo '=== recommendation eligible fetch ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT o.id, o.title, o.organization_name, o.category, o.location,
       o.work_arrangement, o.deadline, o.skills, o.tags,
       o.experience_level, o.career_family, o.relevance_tier,
       o.last_checked_at, o.created_at
FROM opportunities o
WHERE o.status = 'open'
  AND o.verification_status = 'verified'
  AND o.relevance_tier = 'high_confidence_technical'
ORDER BY o.last_checked_at DESC
LIMIT 200;

\echo '=== background_jobs claim ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT id FROM background_jobs
WHERE status IN ('queued', 'retryable') AND run_at <= now()
ORDER BY run_at ASC
FOR UPDATE SKIP LOCKED
LIMIT 1;
SQL
