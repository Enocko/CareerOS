# Universal Opportunity Schema v1

**Date:** 2026-08-30  
**Status:** Implemented (Model C)  
**Migration:** `apps/api/migrations/000010_universal_opportunity_schema_v1.up.sql`

---

## 1. Schema additions

| Column | Type | Nullable | Purpose |
|--------|------|----------|---------|
| `opportunity_type` | `VARCHAR(50)` | NOT NULL (default `employment`) | Canonical opportunity kind (Model C) |
| `type_metadata` | `JSONB` | NOT NULL (default `{}`) | Type-specific validated metadata |
| `verification_method` | `VARCHAR(50)` | NULL | Provenance / trust establishment method |
| `employment_mode` | `VARCHAR(50)` | NULL | Employment schedule mode (`full_time`, `part_time`, `seasonal`) |

`category` is **retained** as a deprecated compatibility alias. New logic must use `opportunity_type`.

---

## 2. Taxonomy (Model C)

### `opportunity_type`

| Value | Meaning |
|-------|---------|
| `employment` | Internships, co-ops, apprenticeships, new-grad and early-career jobs |
| `research` | REU, URP, summer lab programs |
| `scholarship` | Financial award programs |
| `fellowship` | Stipend-based fellowships |
| `program` | Insight, immersion, bridge, leadership programs |
| `event` | Conferences, hackathons (use `type_metadata.event_subtype`) |
| `competition` | Case/data challenges |
| `other` | Residual |

### Employment-only dimensions

Populated only when `opportunity_type = employment`:

| Field | Values |
|-------|--------|
| `experience_level` | `internship`, `co_op`, `apprenticeship`, `new_grad`, `early_career`, `unknown` |
| `employment_mode` | `full_time`, `part_time`, `seasonal` |
| `work_arrangement` | `remote`, `hybrid`, `on_site` |
| `career_family` | Relevance Engine v2 enum |

**Removed from `experience_level`:** `fellowship` (fellowships are an `opportunity_type`, not a seniority band).

---

## 3. Category compatibility

`category` remains on all rows for backward compatibility.

| Legacy `category` | `opportunity_type` | Notes |
|-------------------|-------------------|-------|
| `internship`, `full_time`, `part_time`, `apprenticeship` | `employment` | `category` unchanged |
| `fellowship` | `fellowship` | |
| `scholarship` | `scholarship` | |
| `research` | `research` | |
| `hackathon`, `conference` | `event` | |
| `leadership_program` | `program` | |
| `other` | `employment` | Dev/ingestion default |

**Deprecation strategy:** Stop writing new business logic against `category`. Remove column only after all ingestion, API clients, and browse filters migrate to `opportunity_type`.

**Removal condition:** First non-employment source shipped + frontend Explore tabs + no production client depends on `category` alone.

---

## 4. `verification_method`

Complements `verification_status` (freshness/trust state).

| Value | Meaning |
|-------|---------|
| `official_source` | Documented public API/feed (ATS, USAJobs) |
| `partner` | Partner-provided feed (reserved) |
| `manual_verified` | CareerOS/staff verified official URL |
| `community_verified` | Reviewed community submission (reserved) |

Backfill rules:

- `verified` + `source_id IS NOT NULL` → `official_source`
- `verified` + `source_id IS NULL` → `manual_verified`
- Otherwise → NULL

---

## 5. `type_metadata` validation

Package: `apps/api/internal/opportunitytype/`

- Go struct validation per `opportunity_type`
- Rejects unknown keys per type family
- Validates dates (`YYYY-MM-DD`), booleans, numeric bounds, array limits
- Employment opportunities: `{}` only (no extra keys in v1)

**Write boundaries:** Ingestion upsert validates before INSERT/UPDATE. API clients cannot set arbitrary metadata yet (no public write endpoint).

---

## 6. Migration / backfill

Sequence: add nullable columns → deterministic backfill → NOT NULL + CHECK constraints → index.

### Backfill mapping

Uses legacy `category` (see §3). `employment_mode` derived from `full_time`/`part_time` categories.

### Post-migration catalog (development database, 2026-08-30)

| `opportunity_type` | Count |
|--------------------|------:|
| employment | 192 |
| event | 2 |
| fellowship | 2 |
| research | 1 |
| scholarship | 1 |
| **Total** | **198** |

| `verification_method` | Count |
|-----------------------|------:|
| official_source | 188 |
| NULL | 10 |

Unmapped `opportunity_type`: **0**

Non-employment rows (6) are dev seed records with deterministic category mapping — not ingestion data.

---

## 7. Ingestion compatibility

Central normalization in `ingestion.Repository.UpsertOpportunity`:

- `opportunity_type = employment`
- `verification_method = official_source`
- `employment_mode` from legacy `category` when applicable
- `type_metadata = {}`
- Legacy `category` preserved

Greenhouse, Ashby, Lever, USAJobs unchanged from user perspective.

---

## 8. Recommendation compatibility

`recommendation.Repository.ListCandidates` adds:

```sql
AND o.opportunity_type = 'employment'
```

Scoring unchanged. Non-employment rows cannot enter employment recommendations.

---

## 9. Relevance Engine v2

Unchanged. Employment ingestion continues to populate `career_family`, `experience_level`, `relevance_tier`.

---

## 10. Browse API

- Default behavior unchanged (relevance-tier gating for employment feed)
- Optional filter: `?opportunity_type=employment` (or any valid type)

---

## 11. Saved opportunities

No schema change. `saved_opportunities` remains type-agnostic.

---

## 12. Application tracker boundary

**Employment-only.** Server guard on `POST /api/v1/applications`:

- Rejects `opportunity_type != employment` with HTTP 400

No new statuses. Scholarship/research tracking deferred.

---

## 13. Notifications

Unchanged. Deadline reminders target application-linked employment opportunities via existing job processor. No new notification types for non-employment.

---

## 14. Known limitations

- No non-employment ingestion sources yet
- No public API to write `type_metadata` (validators ready for curation adapter)
- `category` still exposed in API responses
- No type-aware recommendations or Explore tabs
- DB does not enforce employment-only dimensions via CHECK (Go validation only — avoids migration fragility on legacy rows)

---

## 15. Next recommended milestone

**First manual-verified non-employment source:** Grambling career center curation or TMCF open scholarships — using `opportunity_type` + validated `type_metadata`, without ingestion adapter complexity.
