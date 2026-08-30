# ATS Ingestion (Milestone 3)

**Status:** Implemented (Greenhouse, Ashby, Lever v1)  
**Last updated:** 2026-08-29

---

## Provider choice: Greenhouse

Greenhouse was selected as the first ATS provider because it exposes a **documented, unauthenticated public Job Board API**:

```
GET https://boards-api.greenhouse.io/v1/boards/{board_token}/jobs?content=true&page={n}&per_page={n}
```

Lever also offers a public postings API (`api.lever.co/v0/postings/{slug}`), but Greenhouse’s pagination, `content=true` descriptions, and stable numeric job IDs fit the existing ingestion adapter model with minimal transformation.

Lever will be added as a second adapter after Greenhouse is validated end-to-end.

---

## Provider: Ashby (Milestone 4)

Ashby uses the documented public Job Postings API:

```
GET https://api.ashbyhq.com/posting-api/job-board/{board_token}?includeCompensation=true
```

Only `isListed=true` postings are retained. Each board is a separate `opportunity_source` for per-board sync isolation.

---

## Provider: Lever (Milestone 5)

Lever uses the documented public Postings API with `skip`/`limit` pagination:

```
GET https://api.lever.co/v0/postings/{board_token}?mode=json&skip={n}&limit={n}
```

Pagination stops when a page returns fewer than `limit` results (or zero). US API is tried first; EU host is used as fallback on 404.

### Ingestion metrics (all providers)

Each `ingestion_runs` row records:

| Metric | Meaning |
|---|---|
| `records_raw_fetched` | Postings returned by upstream before relevance filtering |
| `records_retained` | Postings kept after relevance (and Ashby public-listing) filters |
| `records_filtered_out` | `raw_fetched - retained` |
| `records_created` / `records_updated` | Upsert outcomes |
| `records_stale` / `records_closed` | Post-sync lifecycle (successful runs only) |

---

## Architecture (modular monolith)

ATS ingestion reuses the existing pipeline:

```
employer_boards (registry)
    → opportunity_sources (one row per board = sync boundary)
        → ingestion adapter (greenhouse)
            → relevance filter (rule-based)
                → normalizer
                    → upsert + post-sync stale/close
```

Each curated employer board is registered as its **own** `opportunity_source`. `Service.RunSource` is called once per enabled source in `RunAll`, giving **per-board failure isolation** — a failed Stripe sync cannot stale Dropbox listings.

---

## Employer registry

`employer_boards` stores maintainable metadata:

| Field | Purpose |
|---|---|
| `employer_name` | Display name (e.g. Stripe) |
| `ats_provider` | `greenhouse` or `lever` (future) |
| `board_token` | Greenhouse board slug |
| `source_url` | Public careers page |
| `tags` | Sector tags (technology, finance, …) |
| `enabled` | Toggle ingestion without deleting history |
| `opportunity_source_id` | FK to sync configuration |

CareerOS does **not** crawl the open web. Only boards explicitly listed in `employer_boards` are ingested.

---

## Relevance filtering

`internal/ingestion/relevance` applies conservative rule-based filtering before upsert:

- **Include:** titles with intern/internship, co-op, new grad, early career, campus
- **Exclude:** recruiter / talent acquisition titles unless they also match intern signals
- **Exclude false positives:** e.g. “Internal Audit” (no `\bintern\b` word boundary match)

Raw source fields are preserved through normalization; filtered-out roles are never upserted.

---

## Verification semantics

Same as USAJobs:

| Status | Meaning |
|---|---|
| `verified` | Seen in a **completed** successful board sync |
| `stale` | Missing from source after N successful syncs |
| `closed` | Past explicit deadline or administratively closed |
| `unverified` | Manual / dev seed |

Failed, partial, or timed-out syncs **never** increment `missed_sync_count` or mark listings stale/closed.

---

## Deduplication

**Within a source:** `(source_id, external_id)` unique index — Greenhouse numeric `id` per board.

**Across sources (future):** The same role may appear on USAJobs and a company board, or on Greenhouse and Lever after both are enabled. Cross-source fuzzy deduplication is **out of scope** for v1. A future approach:

1. Normalize `(organization_name, title, location)` fingerprint
2. Prefer the listing with the most recent `last_seen_at` and authoritative `application_url`
3. Surface “also listed on …” in the UI without hiding either source-backed link until confidence is high

---

## Initial curated boards (verified public tokens)

| Employer | Board token | Tags |
|---|---|---|
| Stripe | `stripe` | technology, fintech |
| Datadog | `datadog` | technology |
| Cloudflare | `cloudflare` | technology, cybersecurity |
| Figma | `figma` | technology, product |
| Discord | `discord` | technology |
| Roblox | `roblox` | technology, gaming |
| Coinbase | `coinbase` | technology, finance |
| Dropbox | `dropbox` | technology |
| Block | `block` | technology, fintech |
| Lyft | `lyft` | technology |

---

## Operations

```bash
make migrate   # applies 000004_employer_boards_greenhouse
make ingest    # syncs USAJobs + all enabled Greenhouse boards
```

Greenhouse adapter settings:

- 45s HTTP timeout per request
- Single request per board (`content=true`; the public API returns the full job list in one response)
- Safe retries on 429/5xx (max 2)
- 4xx (except 429) fail fast without stale side effects
