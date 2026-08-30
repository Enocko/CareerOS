# CareerOS Load Tests (k6)

Reproducible API load tests for Observability + Reliability v1.

## Prerequisites

- PostgreSQL running (`make up`)
- API server running (`make api-run`)
- [k6](https://k6.io/docs/get-started/installation/) installed

## Run baseline profile

```bash
mkdir -p tests/load/results
k6 run tests/load/baseline.js
```

Environment overrides:

```bash
BASE_URL=http://localhost:8080 \
LOAD_EMAIL=loadtest@gram.edu \
LOAD_PASSWORD=loadtest-pass-12345 \
k6 run tests/load/baseline.js
```

## Scenarios covered

1. Opportunity browse (`GET /api/v1/opportunities`)
2. Search/filter (`query` + `category`)
3. Personalized recommendations (`GET /api/v1/opportunities/recommended`)
4. Application dashboard (`GET /api/v1/applications`)
5. Notification unread count (`GET /api/v1/notifications/unread-count`)

Authentication uses a deterministic load-test user (register on first run).

## Output

- Console summary with p50/p95/p99 per workflow
- `tests/load/results/baseline-summary.json` for documentation

## Notes

- Does **not** hit third-party ATS APIs
- Results are environment-specific; record hardware and DB state when reporting
