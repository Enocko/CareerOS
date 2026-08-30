# Research Discovery v1

CareerOS Research Discovery uses a **two-stage trust model** separating NSF award verification from student application availability.

## Critical distinction

| Concept | What it means | CareerOS field |
|---------|---------------|----------------|
| **Source / record verification** | NSF award data is authoritative and current | `verification_status = verified` |
| **Application availability** | Whether a student can apply **now** | `type_metadata.application_status` |

An NSF-funded REU site can be:

- `verification_status = verified` (award confirmed via NSF Award API)
- `application_status = unknown` (no verified student-facing application window)

**These are not the same thing.** CareerOS must never imply “apply now” from award data alone.

---

## 1. NSF Award Discovery (Stage 1)

### Source

**NSF Award API** (`api.nsf.gov/services/v1/awards.json`) — [documented public API](https://www.research.gov/common/webapi/awardapisearch-v1.htm).

### What it proves

- NSF currently funds an REU Site award
- Award ID, title, institution, abstract, award dates
- Official NSF award record URL

### What it does NOT prove

- Current application deadline
- Whether applications are open, upcoming, or closed
- That ETAP accepts applications for this site today
- Stipend, housing, or cycle-specific details unless explicitly in abstract

### Ingestion unit

**One NSF REU Site award = one candidate REU site record** (`external_id` = award ID).

---

## 2. Application Availability (Stage 2)

### Status values (`type_metadata.application_status`)

| Status | Meaning |
|--------|---------|
| `open` | Verified from official student-facing source that applications are currently accepted |
| `upcoming` | Verified that applications will open in the future |
| `closed` | Verified that applications are closed for the current cycle |
| `unknown` | Award verified; application availability not yet established |

### Verification methods (`type_metadata.availability_verification_method`)

| Method | Meaning |
|--------|---------|
| `unknown` | Default after NSF award-only discovery |
| `nsf_award_only` | Status derived only from award data (always `unknown` for applications) |
| `automated_official` | Verified via authorized automated official source |
| `manual_verified` | Staff verified against official program page |
| `partner_verified` | Partner-provided verification |

### Evidence fields (when status is `open` / `upcoming` / `closed`)

- `application_verification_source_url` — official page used
- `application_verified_at` — ISO timestamp of verification

---

## 3. URL rules

| URL type | Field | Rules |
|----------|-------|-------|
| NSF award record | `source_url` | Always `https://www.nsf.gov/awardsearch/showAward?AWD_ID={id}` |
| Program/info page | `type_metadata.program_url` | Institutional REU page from abstract when present |
| Application page | `application_url` | **Only** when a specific application relationship is verified |

**Not sufficient for `application_url`:**

- Generic `https://etap.nsf.gov` homepage
- NSF award search page
- Institution homepage without application path

After availability fix (v1.1): NSF award-only ingest sets `application_url = NULL`.

---

## 4. Deadline integrity

`deadline` is populated **only** when an official student-facing source provides an application deadline.

**Never inferred from:**

- NSF `expDate` (award expiration)
- `startDate` (award start)
- `type_metadata.program_start` / `program_end` (award/program period)

---

## 5. Recurring REU lifecycle

NSF awards often span multiple summers (e.g., 2027–2029). CareerOS models this as:

**One opportunity per award ID** with cycle-specific availability in `application_status`.

When a new award replaces a site, it receives a new `external_id`. The prior record goes `stale` after missed syncs — not `closed` unless application evidence says closed.

Award expiration (`program_end`) does **not** close student applications automatically.

---

## 6. Two-stage pipeline architecture

```
NSF Award API
  → Candidate discovery (nsf_reu adapter)
  → Normalize + validate
  → Upsert with application_status = unknown
  → Availability verification (see RESEARCH_VERIFICATION_WORKFLOW_V1.md)
  → Published with open/upcoming/closed when evidenced
```

Availability verification (v1 implemented):

- `manual_official_page` — admin workflow against institution program pages
- `automated_official` — reserved for authorized feeds
- `partner_verified` — reserved for partnerships

**No generic web crawling.** Heterogeneous program pages require curated/authorized mechanisms.

---

## 7. NSF REU Site Directory investigation

| Question | Finding |
|----------|---------|
| Public URL | `https://www.nsf.gov/funding/initiatives/reu/search` |
| Structured access | JavaScript SPA; no documented public API or export verified in v1 |
| Award number exposed | Not confirmed in machine-readable form |
| Automated retrieval | **Not documented/permitted** — `OFFICIAL_PUBLIC_PAGE` only |
| Recommendation | Use for manual verification workflow; do not reverse-engineer |

---

## 8. Browse / API behavior

- `GET /api/v1/opportunities?opportunity_type=research`
- Optional: `?application_status=open|upcoming|unknown|closed`
- Research browse bypasses employment `relevance_tier`
- Sort: `open` → `upcoming` → `unknown` → `closed`
- `opportunities.status = open` means **catalog visibility**, not “applications open”

### Terminology

| Avoid | Use instead |
|-------|-------------|
| “290 open REUs” | “290 NSF-funded REU site candidates” |
| “Verified opportunity” (ambiguous) | “NSF award verified” + separate application status |
| “Apply now” without `application_url` | “View NSF award” / “View program website” |

---

## 9. Metrics (separate counts)

| Metric | Description |
|--------|-------------|
| Candidate REU sites discovered | Active REU Site awards retained from NSF API |
| Source-verified records | `verification_status = verified` |
| Applications verified open | `application_status = open` |
| Applications upcoming | `application_status = upcoming` |
| Applications closed | `application_status = closed` |
| Availability unknown | `application_status = unknown` |
| Direct application URLs verified | `application_url IS NOT NULL` |
| Program URLs captured | `type_metadata.program_url IS NOT NULL` |

---

## 10. Failure safety

- Failed NSF sync → no stale/close of existing verified awards
- Failed availability verification → record stays `unknown`, not deleted/closed
- Idempotency via `(source_id, external_id)`

---

## 11. Product trust invariant

A student browsing Research must **never** see “apply now” unless `application_status = open` with a verified `application_url`.

Unknown-status records appear with **“Application status unknown”** and explanatory copy.

---

## 12. Live validation (post availability fix)

| Metric | Count |
|--------|------:|
| Candidate REU sites (source verified) | 290 |
| Applications verified OPEN | 0 |
| UPCOMING | 0 |
| CLOSED | 0 |
| UNKNOWN | 290 |
| Direct `application_url` | 0 |
| `program_url` captured | 48 |

---

## 13. Part 0 — Dev seed records (unchanged)

Six dev_seed non-employment records remain `unverified` and hidden from default browse. Grambling CS Research Assistant does not appear in verified research catalog.

---

## 14. Known limitations

1. No automated application availability verification yet
2. NSF Award API cap (3,000 results/query)
3. Program URLs only when present in award abstract
4. NSF REU Directory not integrated (no authorized feed)
5. ETAP per-site listings require separate authorized access

---

## 15. Next milestone

See `docs/RESEARCH_VERIFICATION_WORKFLOW_V1.md` for the implemented verification workflow. Next: scale pilot verifications (~50 queue items) without new discovery sources.
