# Personalized Discovery v1 — Validation Report

**Milestone:** CareerOS Personalized Discovery v1  
**Date:** 2026-08-30  
**Scope:** Deterministic, explainable recommendation ranking (no AI/LLM/embeddings)

---

## 1. Student Profile Fields Used

Existing `student_profiles` fields — **no schema changes required**:

| Field | Use in scoring |
|-------|----------------|
| `major` | Infer career families (e.g. Computer Science → software_engineering) |
| `graduation_year` | Education compatibility signal (PhD mismatch penalty only when explicit) |
| `career_interests` | Infer career families |
| `desired_roles` | Infer career families |
| `skills` + `technologies` | Skills overlap scoring |
| `preferred_locations` | Location preference scoring |
| `work_arrangement` | Work arrangement scoring |
| `experience_level` | Map to opportunity experience levels (`intern` → internship/co-op; `entry` → new_grad/early_career) |

**Not used (not legitimately modeled or unnecessary):**
- Work authorization / visa status
- GPA, demographics
- Saved/applied as positive signals (applied = hard exclude; saved = small ranking penalty)

**Existing opportunity fields used:**
- `career_family`, `experience_level`, `education_level` (Relevance Engine v2)
- `skills`, `location`, `work_arrangement`, `deadline`, `last_checked_at`, `verification_status`

---

## 2. Schema Changes

**None.** Personalized Discovery v1 uses existing profile and opportunity classification columns.

---

## 3. Scoring Factors

| Factor | Type | Description |
|--------|------|-------------|
| Career family | Ranking | Match inferred student families to opportunity `career_family` |
| Experience level | Ranking | Match profile experience to opportunity experience |
| Skills overlap | Ranking | Overlap between profile skills/technologies and opportunity skills/title |
| Work arrangement | Ranking | Match profile preference to opportunity arrangement |
| Location | Ranking | Match preferred locations to opportunity location / remote |
| Freshness | Ranking | Recency of `last_checked_at` (source verification) |
| Deadline urgency | Ranking | Upcoming deadlines score higher; missing deadline = neutral partial credit |
| Already saved | Ranking penalty | −3 points (does not hide opportunity) |
| Education mismatch | Ranking penalty | −10 when opportunity requires PhD and student has undergrad signals only |

---

## 4. Exact Scoring Weights

Centralized in `apps/api/internal/recommendation/config.go`:

| Component | Points |
|-----------|-------:|
| Career family | 30 |
| Experience level | 20 |
| Skills overlap | 20 (scaled by overlap ratio) |
| Work arrangement | 10 |
| Location | 10 |
| Freshness | 5 |
| Deadline urgency | 5 |
| **Maximum (profile-complete)** | **100** |

**Penalties:** already saved −3 · PhD education mismatch −10

**Cold-start weights** (incomplete profile): baseline 35 + freshness (3) + deadline (2) + student relevance (5) + verified (2)

---

## 5. Hard Filters

Conservative exclusions before scoring:

| Filter | Rationale |
|--------|-----------|
| `status != open` | Closed opportunities |
| `verification_status != verified` | Trust model |
| `relevance_tier != high_confidence_technical` | Technical student feed only |
| `deadline < today` (when deadline set) | Expired postings |
| `has_application = true` | Do not recommend duplicates of tracked applications |
| Test fixture external IDs | Same exclusions as browse API |

**Not hard-filtered:** missing education, missing location, ambiguous career family, saved opportunities.

---

## 6. Ranking Signals

All scoring components above are ranking signals except hard filters. Opportunities with partial/unknown metadata receive partial credit (e.g. unknown career family → 15% of career family points).

---

## 7. Cold-Start Behavior

When profile is incomplete or missing:
- Recommendations still returned (never empty solely due to incomplete profile)
- Fallback uses verification freshness, deadline urgency, and student-relevant experience levels
- UI shows nudge: “Complete your profile for better match explanations”
- `meta.profile_complete` returned in API response

---

## 8. Recommendation API

### `GET /api/v1/opportunities/recommended`

Authenticated. Paginated. Does not alter browse/search.

**Query params:** `page`, `per_page`

**Response:**
```json
{
  "data": [{
    "opportunity": { "...Summary fields..." },
    "match_score": 91,
    "factors": [{ "key": "career_family", "label": "Career family", "points": 30, "max": 30 }],
    "match_reasons": ["Software Engineering matches your career interest", "..."],
    "match_summary": "Software Engineering · Python · Internship"
  }],
  "pagination": { "page": 1, "per_page": 10, "total": 90, "total_pages": 9 },
  "meta": {
    "profile_complete": true,
    "has_profile": true,
    "catalog_scored": 90,
    "eligible_count": 90
  }
}
```

### `POST /api/v1/opportunities/recommended/events`

Instrumentation endpoint. Events: `recommendation_impression`, `recommendation_opened`, `opportunity_saved_from_recommendation`, `official_application_link_clicked`

Browse API unchanged: `GET /api/v1/opportunities`

---

## 9. UI Changes

| Change | Path |
|--------|------|
| **For You** page (recommended feed) | `/recommended` — `RecommendedPage.tsx` |
| **Browse** preserved | `/opportunities` — unchanged search/filter/pagination |
| Nav split | “For You” + “Browse” in navbar |
| Match display | Score %, compact chips, expandable “Why this matches” |
| Actions | Save, View & apply; detail page tracks recommendation events |

---

## 10. Example Recommendation Explanations

**CS student, Python, remote preference — Software Engineer Intern:**
- Match: **~91%**
- Why: Software Engineering matches your career interest · Internship matches your experience level · Python overlaps with your skills · Remote option matches your preference · Recently verified from source
- Compact: `91% match · Software Engineering · Python · Internship`

**Incomplete profile — cold start:**
- Match: **~42%**
- Why: Student-relevant opportunity · Recently verified from source · Complete your profile for better matches

---

## 11. Evaluation Fixture Results

`fixture_eval_test.go` — relative ranking assertions pass:

| Profile | Expected top rank |
|---------|-------------------|
| CS / SWE student | Software Engineer Intern > Technology Analyst Intern |
| Data science student | Data Science Intern > Software Engineer Intern |
| Cybersecurity student | Security Engineering Intern > Product Management Intern |
| Incomplete profile | Software Engineer Intern > Staff Engineer |

---

## 12. Tests

| Suite | Status |
|-------|--------|
| `recommendation/scorer_test.go` | PASS — profiles A–F behaviors, boundaries, determinism |
| `recommendation/fixture_eval_test.go` | PASS — relative ranking fixture |
| `recommendation/latency_test.go` | PASS — latency under 2s threshold |
| `tests/integration/opportunities_recommended_test.go` | PASS — API ranking + applied exclusion |

Full `go test ./...` — PASS  
`npm run build` — PASS

---

## 13. Current Catalog Size

- **Technical feed candidates:** 90 (`high_confidence_technical`, verified, open)
- **Total verified catalog:** 188 (includes ambiguous/non-technical not in recommendation feed)

---

## 14. Measured Recommendation Latency

Measured via `TestRecommendationLatency` against live PostgreSQL catalog:

| Metric | Value |
|--------|------:|
| Catalog scored | 90 |
| Eligible after hard filters | 90 |
| End-to-end service latency | **13 ms** |
| HTTP handler latency (integration log) | **6–8 ms** |

**Work performed:** Single SQL query for candidates + in-memory deterministic scoring/sort for ~90 rows. No Redis, no vector search.

---

## 15. Recommendation Quality Concerns

1. **Career interest mapping is keyword-based** — free-form `career_interests` / `desired_roles` mapped to `career_family` via deterministic rules; controlled vocabulary on profile would improve precision later.
2. **Profile ↔ opportunity experience enum mismatch** — profile uses `intern`/`entry`; opportunities use `internship`/`new_grad`; explicit mapping layer added but may need UI alignment.
3. **`Technology Analyst` roles** — may rank for finance/compliance students with tech keywords; partial credit only, not hard-filtered.
4. **Saved penalty is mild (−3)** — saved roles can still appear; students use Saved page for explicit bookmarks.
5. **In-memory scoring** — acceptable at current catalog size (~90–200); monitor if catalog grows past low thousands.

---

## Architecture

```
Student Profile (existing)
        +
Opportunity Classification (Relevance v2)
        ↓
recommendation.Service
        ↓
Deterministic Scorer (config.go weights)
        ↓
Ranked Opportunities
        ↓
Explanation Generator (explain.go)
        ↓
API (/opportunities/recommended) + UI (/recommended)
```

**Key files:**
- `apps/api/internal/recommendation/` — scorer, config, service, handler
- `apps/web/src/pages/RecommendedPage.tsx`
- `apps/web/src/pages/OpportunitiesPage.tsx` (browse preserved)
