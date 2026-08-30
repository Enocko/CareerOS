# Research Verification Workflow v1

CareerOS converts NSF REU **candidates** (verified award records) into **actionable application availability** through a lightweight internal verification workflow.

## 1. Candidate vs actionable opportunity

| Layer | Meaning | Example |
|-------|---------|---------|
| **Candidate (Stage 1)** | NSF funds this REU site | `verification_status = verified`, `application_status = unknown` |
| **Actionable (Stage 2)** | Student can act on a specific cycle | `application_status = open` with verified `application_url` |

NSF award ingestion never implies applications are open.

## 2. Program / cycle model (Model C)

CareerOS uses **one `opportunities` row per NSF award ID** (`external_id`). Recurring REU programs do not get separate rows per cycle.

Current-cycle fields live on the opportunity:

- `type_metadata.application_status`
- `type_metadata.cycle_label`, `opens_at`
- `type_metadata.application_verified_at`, `next_verification_at`
- `application_url`, `deadline` (top-level, cycle-specific)

Re-verification **overwrites** current cycle state. Historical cycles are preserved in `research_availability_verifications` (append-only audit).

### Why not separate program + cycle tables?

- Smallest model that fits existing universal opportunity schema
- Saves and reminders reference stable opportunity IDs across annual cycles
- No duplicate NSF award rows
- Trade-off: only **current** cycle is student-facing; history is in audit table

## 3. Verification workflow

```
NSF candidate (unknown availability)
  → prioritized queue
  → reviewer opens official student-facing source
  → records evidence via POST /api/v1/admin/research/opportunities/{id}/verify
  → opportunity updated + audit row inserted
  → next_verification_at scheduled
```

Admin-only. Students cannot submit verifications.

## 4. Verification evidence

Each verification creates a row in `research_availability_verifications`:

| Field | Purpose |
|-------|---------|
| `application_status` | open / upcoming / closed / unknown |
| `application_url` | Specific application destination (required for open) |
| `verification_source_url` | Official page used as evidence |
| `opens_at`, `deadline` | Cycle dates from official source only |
| `cycle_label` | e.g. "Summer 2027" |
| `verification_method` | How availability was determined |
| `verified_at`, `verified_by` | Audit timestamp and reviewer user ID |
| `next_verification_at` | Scheduled re-check |
| `notes` | Optional reviewer notes |

Structured log on apply: `opportunity_id`, `previous_status`, `new_status`, `verification_method`, `verified_at`.

## 5. Status definitions

| Status | Semantics |
|--------|-----------|
| **open** | Official source currently accepts applications |
| **upcoming** | Official source confirms a future window; not yet open |
| **closed** | Official source confirms the relevant cycle is closed/expired |
| **unknown** | CareerOS cannot establish application availability |

Never infer status from NSF award activity dates.

## 6. Deadline rules

- Populate `deadline` only from authoritative student-facing sources
- Never use NSF award expiration, award start, or program end dates
- If unverified: `deadline = NULL`
- Server rejects `closed` with a future deadline

## 7. Application URL rules

`application_url` must be a **specific** application destination (institutional REU form, specific ETAP listing, official portal).

Rejected:

- Generic ETAP homepage (`https://etap.nsf.gov`)
- Generic university homepage
- NSF award record URL

Program information belongs in `type_metadata.program_url` or `source_url`.

## 8. Verification methods

| Method | Use |
|--------|-----|
| `manual_official_page` | Reviewer checked official program page |
| `automated_official` | Authorized automated official source (future) |
| `partner_verified` | Partner-provided (future) |
| `unknown` | No official page verified yet |
| `nsf_award_only` | Ingestion default; availability remains unknown |

## 9. Re-verification policy

`ComputeNextVerification()` defaults:

| Status | Next check |
|--------|------------|
| **open** | 7 days before deadline, or within 14 days (whichever is sooner) |
| **upcoming** | 3 days before `opens_at`, or 7 days |
| **closed** | 180 days (next cycle season) |
| **unknown** | 90 days |

Overdue `next_verification_at` increases queue priority (+40 score). No separate scheduler; queue and metrics surface stale records.

## 10. Student-facing visibility

Research page (`/research`) sections:

1. **Applications open now** — verified `open` only
2. **Upcoming** — verified `upcoming`
3. **Research programs (availability unknown)** — NSF candidates; visually distinct (dashed border), not labeled as apply-now

Search spans all research records with `application_status` filter support.

## 11. Save semantics (recurring programs)

Students save the **opportunity ID** (NSF award / program identity). When a new cycle is verified:

- Same saved record points to updated cycle fields
- Deadline and `application_url` reflect the current verification
- Reminders use the new deadline only when `application_status = open`

No per-cycle save rows required.

## 12. Deadline reminders

`ListSavedDeadlineCandidates` reminds research opportunities **only when** `application_status = open` and `deadline` is set.

- OPEN + saved + deadline → 7/3/1-day reminders (existing notification job)
- UNKNOWN → no reminder
- CLOSED / UPCOMING without open status → no reminder

## 13. Admin / internal workflow

Routes (require auth + `CAREEROS_ADMIN_EMAILS`):

- `GET /api/v1/admin/research/queue` — prioritized candidates
- `GET /api/v1/admin/research/metrics` — catalog counts
- `GET /api/v1/admin/research/opportunities/{id}` — candidate detail
- `GET /api/v1/admin/research/opportunities/{id}/verifications` — audit history
- `POST /api/v1/admin/research/opportunities/{id}/verify` — apply verification

Minimal UI: `/admin/research` (protected route; 403 from API if not admin).

## 14. Prioritization queue

Deterministic score (higher = verify first):

| Signal | Points |
|--------|--------|
| `next_verification_at` overdue | +40 |
| `program_url` known | +30 |
| Never application-verified | +20 |
| `application_status = unknown` | +10 |

Tie-break: `last_seen_at DESC`.

No personalization. No AI.

## 15. Pilot verification results (Aug 2026)

15 NSF candidates verified via `apps/api/cmd/research-pilot` using official program pages.

| Status | Count | Notes |
|--------|-------|-------|
| closed | 4 | JMU Math, UNLV MOE, Danforth, (Cal Academy 2026 superseded) |
| upcoming | 1 | Cal Academy Summer 2027 (opens ~Dec 2026) |
| open | 0 | No defensible open applications in pilot window |
| unknown | 10 | Official page found; cycle status not confirmed |

Verified deadlines: 3 (JMU, UNLV, closed cycles). Verified application URLs: 0 (no open status in pilot).

## 16. Limitations

- No ETAP scraping or web crawler
- No bulk automation of 290 candidates
- Current cycle only on opportunity row; historical cycles in audit table
- Admin UI is minimal (queue + form)
- Re-verification is queue-driven, not push-notified to reviewers
- OPEN count may be zero between REU recruiting seasons

## 17. Recommended next milestone

**Research verification scale-up v1**: Expand pilot to ~50 high-priority queue items, add optional `closed` browse section, and evaluate lightweight ETAP deep-link verification for programs that publish specific listing URLs (still no scraping).
