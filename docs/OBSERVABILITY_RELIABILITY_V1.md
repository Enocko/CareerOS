# Observability + Reliability & Load Validation v1

**Date:** 2026-08-30  
**Status:** Complete — ready for review

## Architecture

CareerOS observability remains **in-process and PostgreSQL-backed**:

- JSON structured logs (`slog`) with `service` field per process (`api`, `worker`, `scheduler`)
- HTTP middleware: `request_id`, latency, status, `outcome`
- Prometheus-compatible `GET /metrics` (no Grafana/Prometheus server added)
- Liveness (`/health`) separated from readiness (`/ready`)
- Domain gauges refreshed on metrics scrape from existing SQL repositories
- Background job lease recovery for abandoned `processing` jobs

No Redis, Kafka, tracing infrastructure, or new services were introduced.

---

## 1. Observability changes

| Area | Change |
|------|--------|
| HTTP logging | Added `outcome` (`success` / `client_error` / `error`) |
| Request ID | Existing `X-Request-ID` propagation retained; included in recommendation logs |
| Service identity | `observability.SetDefault(service, level)` for api/worker/scheduler |
| Recommendation logs | Added `request_id`, `outcome`; Prometheus histogram |
| Job worker | Reclaims stale `processing` jobs; logs `outcome` on completion/failure |
| Health model | `/health` = liveness only; `/ready` = PostgreSQL ping |

**Not logged:** passwords, tokens, secrets, full profile payloads, notification message bodies in new code paths.

---

## 2. Metrics exposed

`GET /metrics` (Prometheus text format):

| Metric | Description |
|--------|-------------|
| `careeros_http_requests_total{method,route,status_class}` | Request count |
| `careeros_http_request_duration_seconds{method,route}` | Latency histogram |
| `careeros_http_requests_in_flight` | Active requests |
| `careeros_background_jobs_*` | Queued/processing/retryable/failed/completed, retries, oldest queued age |
| `careeros_notifications_created_total` | Total notifications |
| `careeros_ingestion_success_total` / `failure_total` | Latest run outcome per enabled source |
| `careeros_recommendation_duration_seconds` | Recommendation handler latency |
| Go/process collectors | Runtime stats |

---

## 3. Load-test environment

| Item | Value |
|------|-------|
| Machine | Apple M3, 16 GB RAM, arm64 |
| OS | macOS (darwin 25.5.0) |
| API | Go 1.25, local `go run ./cmd/server` |
| Database | PostgreSQL 16 in Docker (`localhost:5433`) |
| Catalog size | ~198 opportunities (~90 technical feed) |
| Tool | k6 via `grafana/k6` Docker image |
| Auth | Deterministic user `loadtest@gram.edu` (register-on-first-run) |

---

## 4. Load scenarios

Script: `tests/load/baseline.js`

Each virtual user iteration executes:

1. Browse (`GET /api/v1/opportunities?per_page=20`)
2. Search/filter (`query=engineer&category=internship`)
3. Recommendations (`GET /api/v1/opportunities/recommended`)
4. Applications list (`GET /api/v1/applications`)
5. Notification unread count (`GET /api/v1/notifications/unread-count`)

Ramp profile: **1 → 10 → 25 → 50 → 100 VUs** over 2.5 minutes, hold, ramp down (3 minutes total).

---

## 5. p50 / p95 / p99 by major endpoint

### Full baseline (100 max VUs, 3 minutes)

Measured 2026-08-30 against API with observability changes on port 8081.

| Workflow | p50 | p95 | p99 | Notes |
|----------|-----|-----|-----|-------|
| HTTP overall | — | **14.6 ms** | — | k6 trend `http_req_duration` |
| Browse | — | **19.7 ms** | — | |
| Search/filter | — | **16.3 ms** | — | |
| Recommendations | — | **3.1 ms** | — | In-memory scoring remains fast under load |
| Applications | — | **13.0 ms** | — | |
| Notifications | — | **2.3 ms** | — | |

### Supplemental (50 VUs, 45 seconds)

| Workflow | p50 | p95 |
|----------|-----|-----|
| Browse | 4.8 ms | 30.4 ms |
| Search/filter | 5.6 ms | 31.5 ms |
| Recommendations | 7.0 ms | 35.0 ms |
| Applications | 4.3 ms | 25.5 ms |
| Notifications | 2.1 ms | 15.2 ms |
| HTTP overall | 4.6 ms | 28.2 ms |

p99 not reported for 100-VU run (k6 summary export used `med` + `p(95)` buckets). Supplemental run max latencies stayed below 620 ms (isolated spikes).

---

## 6. Throughput / error rates

| Profile | Requests | Duration | Throughput | Error rate |
|---------|----------|----------|------------|------------|
| 100 VU ramped | 123,135 | 180 s | ~684 req/s | **0%** |
| 50 VU fixed | 45,675 | 46 s | ~994 req/s | **0%** |

All workflow checks (`browse 200`, `search 200`, etc.) passed with zero failures.

**No scale claims** — these numbers apply only to this local environment and catalog size.

---

## 7. Database / query findings

`make query-analysis` (`scripts/reliability/query-analysis.sh`):

| Query | Plan | Execution time |
|-------|------|----------------|
| Browse COUNT | Index-friendly filters on `status` + `verification_status` | ~2 ms |
| Browse LIST | `Index Scan` on `opportunities_deadline_idx` | **0.10 ms** |
| Recommendation eligible fetch | `Seq Scan` on ~198 rows | **0.13 ms** |
| Job claim (`SKIP LOCKED`) | `Seq Scan` on small `background_jobs` table | **0.03 ms** |

PostgreSQL connection pool: **MaxConns=10**, MinConns=2 (unchanged — no pool exhaustion observed at tested load).

---

## 8. Index / query optimizations

**No new indexes added.**

| Candidate | Evidence | Decision |
|-----------|----------|----------|
| Recommendation `relevance_tier` index | Seq scan over 90 matching rows = 0.13 ms | **Skip** — measurement does not justify index maintenance |
| Browse deadline ordering | Already uses `opportunities_deadline_idx` | No change |

---

## 9. Recommendation performance

| Condition | Latency |
|-----------|---------|
| Single-request baseline (v1) | ~13 ms service time |
| 100 VU load p95 | **3.1 ms** handler-side trend |
| 50 VU load p50 / p95 | 7.0 ms / 35.0 ms |

**Conclusion:** SQL fetch → in-memory score → sort remains sufficient at current catalog size. **Redis not indicated.**

---

## 10. Queue stress results

Test: `TestQueueStressConcurrentClaims` (`internal/jobs/stress_test.go`)

| Metric | Result |
|--------|--------|
| Jobs enqueued | 40 |
| Workers | 4 |
| Jobs completed | 40 |
| Duplicate claims | **0** |
| Drain time | < 5 s |
| Failed/retried | 0 |

---

## 11. PostgreSQL outage results

Script: `scripts/reliability/postgres-outage.sh`

| Step | Observed |
|------|----------|
| DB up | `/health` → 200 `{"status":"ok"}`; `/ready` → 200 connected |
| DB stopped | `/health` → **200** (liveness unchanged); `/ready` → **503** `not_ready` |
| DB restored | `/ready` → 200 connected within ~5 s |

API does not silently report ready when PostgreSQL is unavailable.

---

## 12. Worker-kill / recovery results

**Reliability defect found:** jobs stuck in `processing` after worker crash were never reclaimed.

**Fix:** `ReclaimStaleProcessing` — jobs in `processing` with `locked_at` older than `ProcessingLease` (default **15 min**) return to `retryable`.

Test: `TestReclaimStaleProcessing` — stale job reclaimed and re-claimed successfully.

---

## 13. Provider-failure results

Existing ingestion test coverage (`TestFailedSyncDoesNotMarkStaleOrClosed`, isolated GH fail-board):

- Source run records failure
- Verified opportunities **remain verified** (not stale/closed)
- Other sources unaffected
- API remains healthy (no ATS dependency in readiness)

---

## 14. Duplicate-work results

| Scenario | Result |
|----------|--------|
| Duplicate ingest job enqueue | Suppressed (active idempotency) |
| Duplicate reminder job enqueue | Suppressed while active |
| Duplicate notification insert | Suppressed (`ON CONFLICT`) |
| User-visible duplicate notifications | **0** (integration + unit tests) |

---

## 15. Slow-dependency results

- Ingestion HTTP client timeout: **90 s** (`cmd/worker/main.go`)
- Simulated slow HTTP server with **500 ms** client timeout → error returned promptly; other jobs remain claimable (`TestSlowDependencyTimesOutAndAllowsOtherWork` pattern in worker/http tests)
- Worker does not hang indefinitely; failures go through retry/backoff path

---

## 16. Reliability bugs discovered

| Bug | Severity |
|-----|----------|
| Abandoned `processing` jobs never reclaimed | **High** — fixed |

---

## 17. Fixes implemented

| Fix | Before → After |
|-----|----------------|
| Stale job lease recovery | Stuck `processing` forever → reclaimed to `retryable` after 15 min |
| `/health` vs `/ready` | DB failure made `/health` 503 → liveness always 200; readiness reflects DB |
| Prometheus `/metrics` | No app metrics → HTTP + domain gauges |
| Structured logging | Inconsistent → `service`, `outcome`, `request_id` on key paths |

---

## 18. Remaining bottlenecks / limitations

1. **DB pool hardcoded at 10** — no exhaustion observed at 100 VU / 0% errors; revisit if production concurrency grows.
2. **Metrics scrape queries DB** — acceptable at current scale; not suitable for very high scrape rates.
3. **Notification list pagination total** — still page-count only (pre-existing).
4. **k6 p99** — not exported in default summary trend stats; use JSON export for deeper analysis.
5. **Long ingest occupies one worker slot** — unchanged; monitor as source count grows.

---

## Reproduce commands

```bash
# Environment
make up
make api-run                    # or API_PORT=8081 make api-run

# Tests
make api-test
make web-build

# Load test (requires Docker for k6 image)
make load-test
# or: BASE_URL=http://host.docker.internal:8081 make load-test

# Query analysis
make query-analysis

# Failure experiment: PostgreSQL outage
BASE_URL=http://localhost:8081 make reliability-postgres-outage

# Metrics
curl http://localhost:8081/metrics
curl http://localhost:8081/health
curl http://localhost:8081/ready
```

---

## Test / build results

```
go test ./...     → PASS
npm run build     → PASS
```

New tests: `reclaim_test.go`, `stress_test.go`, health/readiness/metrics integration tests.

---

## Scope guardrails respected

No Redis, Kafka, NATS, RabbitMQ, Kubernetes, Elasticsearch, new ATS, AI/LLM, or product features added.
