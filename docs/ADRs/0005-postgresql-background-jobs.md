# ADR 0005: PostgreSQL-Backed Background Jobs

**Status:** Accepted  
**Date:** 2026-08-30  
**Context:** CareerOS Background Jobs & Notifications v1

## Context

CareerOS needs:

- Scheduled opportunity ingestion without manual `make ingest`
- Application and saved-opportunity deadline reminders
- Durable, retryable, idempotent background work
- Observability at current scale (~50 sources, ~90–200 opportunities, single-region deployment)

The system is a modular monolith with PostgreSQL already as the system of record.

## Decision

Use a **PostgreSQL-backed job queue** with:

- `background_jobs` table for durable work items
- `FOR UPDATE SKIP LOCKED` for worker claim safety
- Partial unique index on active `idempotency_key` values
- Exponential backoff with jitter for retries
- Separate **scheduler** and **worker** processes in the same Go codebase

In-app notifications are stored in a `notifications` table; reminder jobs create notifications idempotently.

## Alternatives Considered

| Option | Why not (for v1) |
|--------|------------------|
| **Redis / BullMQ** | New infrastructure dependency; team must operate another service; catalog size does not require in-memory queue throughput |
| **Kafka / NATS** | Event streaming is overkill for periodic ingest + daily reminders; operational cost exceeds benefit |
| **Temporal** | Heavy operational and conceptual overhead for two job types |
| **Cron-only (`make ingest` in crontab)** | No unified retry/dedup model for reminders; harder overlap protection across hosts |
| **In-process goroutines in API server** | Couples background work to HTTP process lifecycle; poor restart isolation |

## Consequences

**Positive**

- Reuses existing PostgreSQL backups, migrations, and tooling
- At-least-once execution with idempotent handlers is straightforward
- Scheduler and worker scale as separate processes/containers without new repos
- Job metrics queryable via SQL (`jobs-report`)

**Negative**

- Job throughput limited by PostgreSQL polling (acceptable at current scale)
- Workers must be idempotent (explicit design requirement)
- Long-running ingestion blocks one worker slot (mitigated by per-source isolation and overlap guards)

## Implementation Notes

- Job types: `ingest_source`, `deadline_reminder`
- Active idempotency: partial unique index `WHERE status IN ('queued','processing','retryable')`
- Ingest overlap: skip enqueue when source has running `ingestion_runs` row or active ingest job
- Reminder idempotency: `reminder:{user}:{opportunity}:{application|saved}:{deadline}:{window_days}`
- Not exactly-once; duplicate delivery prevented at notification layer via unique `idempotency_key`
