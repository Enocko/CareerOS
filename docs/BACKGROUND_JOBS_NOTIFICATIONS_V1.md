# Background Jobs & Notifications v1 — Validation Report

**Date:** 2026-08-30  
**Status:** Complete — ready for review

## 1. Architecture selected

**PostgreSQL-backed durable job queue** with separate scheduler and worker processes in the same Go modular monolith.

See [ADR 0005](./ADRs/0005-postgresql-background-jobs.md).

## 2. Why it was selected

- PostgreSQL is already the system of record; no new infrastructure (Redis, Kafka, Temporal, etc.)
- Current scale (~50 sources, ~200 opportunities) does not justify a distributed queue
- `FOR UPDATE SKIP LOCKED` provides safe multi-worker claiming
- Partial unique indexes give active-job deduplication without blocking future re-runs after completion
- Scheduler/worker separation matches product boundaries (enqueue vs execute)

## 3. Schema changes

Migration `000009_background_jobs_notifications`:

| Table | Purpose |
|-------|---------|
| `background_jobs` | Durable work queue with status, retries, idempotency, locking |
| `notifications` | In-app student notifications with unique `idempotency_key` |

Indexes:

- Partial unique on `background_jobs(idempotency_key)` for active statuses
- Claim index on `run_at` for queued/retryable jobs
- User notification indexes for list and unread queries

## 4. Job lifecycle

```
queued → processing → completed
                   ↘ retryable → (backoff) → processing
                   ↘ failed (max attempts exhausted)
```

Job types:

- `ingest_source` — runs existing ingestion service for one source
- `deadline_reminder` — creates in-app notification idempotently

## 5. Retry/backoff policy

| Setting | Default |
|---------|---------|
| Max attempts | 5 |
| Base backoff | 30s |
| Max backoff | 30m |
| Strategy | Exponential × 2^(attempt-1) + up to 25% jitter |
| Worker poll | 2s |
| Scheduler tick | 60s |

Permanently invalid payloads still fail after bounded attempts; errors stored in `last_error` (no secrets).

## 6. Idempotency strategy

| Layer | Key pattern |
|-------|-------------|
| Active ingest job | `ingest:source:{source_id}` |
| Reminder job (active) | `reminder:{user}:{opportunity}:{app\|saved}:{deadline}:{window}d` |
| Notification | Same as reminder key on `notifications.idempotency_key` (unique, permanent) |

**At-least-once** job execution; duplicate notification delivery prevented at the notification insert (`ON CONFLICT DO NOTHING`).

Completed/failed jobs release the active idempotency slot so future schedule cycles can re-enqueue when appropriate; notification uniqueness prevents duplicate reminders.

## 7. Scheduled ingestion behavior

- Scheduler calls `ListDueSources` — sources past `sync_interval_minutes` without a running `ingestion_runs` row
- Skips enqueue when:
  - Source has running ingestion run (`HasRunningIngestion`)
  - Active ingest background job exists (`HasActiveIngestJob`)
- Worker delegates to existing `ingestion.Service.RunSource`
- Worker re-checks overlap before running
- Per-source isolation preserved; one source failure does not block others
- Existing ingestion stale/closed safety rules unchanged (reuses ingestion service)

Default source sync interval remains per-source (`sync_interval_minutes`); scheduler does not run all sources aggressively.

## 8. Deadline reminder behavior

**Policy:** reminders at **7, 3, and 1 days** before deadline (configurable via `jobs.Config.ReminderWindows`).

| Candidate | Query rules |
|-----------|-------------|
| Application | `opportunities.deadline` matches target date, deadline NOT NULL, status `open`, application not rejected/withdrawn/closed |
| Saved (not applied) | Same deadline rules, saved row exists, no application for that student+opportunity |

**Never:**

- Invent deadlines (NULL deadline → excluded by SQL)
- Remind for closed opportunities
- Send saved-opportunity “apply soon” when student already has an application

Changing a deadline creates a new idempotency key (different date), so a new reminder window is scheduled safely.

## 9. Notification API

Authenticated routes under `/api/v1/notifications`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/notifications` | Paginated list + unread count |
| GET | `/notifications/unread-count` | Unread badge count |
| PATCH | `/notifications/{id}/read` | Mark one read |
| POST | `/notifications/mark-all-read` | Mark all read |

## 10. Notification UI

- `NotificationBell` in navbar — polls unread count every 60s
- `/notifications` page — simple list, mark read, mark all read, navigate to opportunity/application
- No complex notification center (per scope)

## 11. Concurrency test results

| Test | Result |
|------|--------|
| Two workers claim one job (`FOR UPDATE SKIP LOCKED`) | Pass |
| Duplicate job enqueue (active idempotency) | Pass |
| Overlapping ingest enqueue (running run + active job) | Pass |
| Duplicate notification create | Pass |
| Scheduler duplicate tick (same reminder window) | Pass |

## 12. Failure/retry test results

| Test | Result |
|------|--------|
| Transient failure → retryable status | Pass (worker integration) |
| Retry exhaustion → `failed` | Pass (`TestRetryExhaustionMarksFailed`) |
| Successful completion | Pass |

## 13. Duplicate suppression results

| Scenario | Result |
|----------|--------|
| Re-enqueue active ingest job | Suppressed |
| Re-run reminder job after completion | Allowed; notification insert suppressed |
| Same idempotency key on notification | Suppressed |
| Scheduler re-tick same window | Active job suppressed |

## 14. Operational metrics/reporting

`make jobs-report` prints:

- Queued / processing / retryable / failed / completed counts
- Total retry attempts
- Oldest queued job age
- Notifications created (total)
- Last successful ingestion timestamp per enabled source

Structured JSON logs: job created, claimed, completed, retry scheduled, permanently failed, reminder created, duplicate suppressed, scheduled ingestion started/completed.

## 15. Developer commands

```bash
make up              # PostgreSQL + migrations
make api-run         # API server
make worker          # background job worker
make scheduler       # continuous scheduler
make scheduler-once  # single tick (cron-friendly)
make jobs-report     # queue health report
make web-run         # frontend
```

Local process layout:

1. `make up`
2. `make api-run` (terminal 1)
3. `make worker` (terminal 2)
4. `make scheduler` (terminal 3)
5. `make web-run` (terminal 4)

## 16. Full test/build results

```
go test ./...     → PASS (all packages)
npm run build     → PASS (web)
```

Key test files:

- `internal/jobs/repository_test.go` — enqueue, claim, retry, completion
- `internal/jobs/reminder_policy_test.go` — 7-day reminders, closed/no-deadline exclusion, saved+applied exclusion, ingest overlap
- `internal/notifications/repository_test.go` — notification idempotency
- `tests/integration/notifications_test.go` — API flow end-to-end

## 17. Architectural concerns discovered

1. **List pagination total** — notification list `pagination.total` returns page item count, not DB total (acceptable v1 limitation).
2. **Job idempotency vs notification idempotency** — after a job completes, the same reminder can be re-enqueued; notification uniqueness is the final dedupe gate. This is intentional for at-least-once processing.
3. **Worker slot during long ingest** — one worker is occupied per running ingest job; acceptable at current source count; monitor if source count grows significantly.
4. **No email channel** — in-app only for v1; email not required per milestone scope.

## Scope guardrails respected

No Kafka, NATS, RabbitMQ, Redis, Temporal, Kubernetes, microservices, new ATS, AI/LLM, push/SMS, or additional recommendation sophistication.

Existing product behavior preserved: auth, profile, browse, personalized discovery, save, applications, relevance, ingestion reliability.
