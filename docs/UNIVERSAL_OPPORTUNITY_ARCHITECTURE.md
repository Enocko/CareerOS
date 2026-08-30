# Universal Opportunity Model + Source Expansion Architecture

**Date:** 2026-08-30  
**Status:** Model C implemented in Schema v1 (`000010_universal_opportunity_schema_v1`) — see [`UNIVERSAL_OPPORTUNITY_SCHEMA_V1.md`](UNIVERSAL_OPPORTUNITY_SCHEMA_V1.md)  
**Scope:** Design only. No schema migrations, integrations, or product changes in this milestone.

Machine-readable source research: [`scripts/opportunity_source_research.json`](../scripts/opportunity_source_research.json)

---

## 1. Executive summary

CareerOS is **closer to a universal opportunity model than its product positioning suggests**. The initial `opportunities.category` enum already includes scholarship, research, fellowship, hackathon, conference, apprenticeship, and leadership programs — not only jobs. However, the **runtime system is employment-centric**: ingestion adapters are ATS/government job boards, relevance v2 targets technical student employment, recommendations score career family/skills/work arrangement, and the application tracker models interview pipelines.

**Recommendation:** Evolve incrementally via **additive schema + metadata**, not a rewrite.

| Question | Answer |
|----------|--------|
| Block production employment pilot? | **No** — current model works for verified employment ingestion |
| Modify schema before universal expansion? | **Yes — minimally** (type normalization, source provenance, optional org typing) |
| When? | **After initial pilot** for full universal model; **before first non-employment source** for minimal additive fields |
| First non-employment integration | **Manual verified curation of HBCU-relevant scholarships** (TMCF/UNCF/Grambling career center), then **NSF REU directory curation** |
| Do not integrate now | LinkedIn, Indeed consumer scraping, Jobright, Simplify scraping, Handshake without institutional partnership |

---

## 2. Current model audit

### 2.1 `opportunities` table (actual schema)

| Field | Purpose today | Universal fit |
|-------|---------------|---------------|
| `id` | Primary key | CORE |
| `organization_id` | FK to organizations | CORE (unused by ingestion today) |
| `organization_name` | Display publisher | CORE |
| `title`, `description` | Core content | CORE |
| `category` | Opportunity kind enum | **Partial `opportunity_type`** (see §3) |
| `location` | Geographic text | CORE |
| `work_arrangement` | remote/hybrid/on_site | TYPE-SPECIFIC (employment) |
| `deadline` | Application deadline | CORE (when known) |
| `start_date` | Program/job start | CORE (often `starts_at`) |
| `eligibility` | Free-text eligibility | CORE |
| `skills` | TEXT[] | DERIVED / employment-biased |
| `compensation` | Pay/stipend text | TYPE-SPECIFIC |
| `application_url`, `source_url` | Official links | CORE |
| `source` | Source display name string | DERIVED (denormalized) |
| `source_id`, `external_id` | Ingestion identity | CORE |
| `verification_status` | verified/unverified/stale/closed | CORE (trust) |
| `status` | open/closed | CORE |
| `first_seen_at`, `last_seen_at`, `last_checked_at` | Freshness | CORE |
| `missed_sync_count` | Staleness signal | DERIVED (sync-based sources) |
| `experience_level` | Student career stage | TYPE-SPECIFIC (employment) |
| `career_family` | Technical classification | DERIVED (employment browse) |
| `education_level` | UG/grad/PhD | CORE |
| `relevance_tier` | Browse feed gate | DERIVED (employment product policy) |
| `classification_reasons` | Explainability | DERIVED |

**Not present:** `opens_at`, `ends_at`, `opportunity_type` (distinct from category), `verification_method`, `fields_of_study`, structured stipend/award amount, research area, event dates.

### 2.2 Assumptions audit (code-backed)

| Assumption | Actually true? | Evidence |
|------------|----------------|----------|
| Every opportunity is a job | **No** — category enum includes scholarship, research, etc. | `000001_initial_schema.up.sql` category CHECK |
| Every opportunity has employment type | **No** — nullable; only employment adapters populate it | Ingestion adapters |
| Every opportunity has work arrangement | **Partially** — DB default `on_site`; normalizer defaults on_site | `normalizer.go` |
| Every opportunity belongs to an employer | **De facto yes** — `organization_name` from employer board config | GH/Ashby/Lever adapters |
| Every opportunity has salary | **No** — `compensation` nullable | Schema |
| Every opportunity has an ATS | **No** — USAJobs + manual exist | `opportunity_sources.adapter` |
| Every opportunity can become an application | **De facto yes** — applications FK to opportunities generically | Schema allows any opp |
| Same lifecycle for all types | **No in schema, yes in product** — interview statuses are employment-specific | `applications.current_status` |
| `organizations` is canonical publisher | **No** — `organization_id` never set by ingestion | Ingestion repository |
| Cross-source dedup exists | **No** — only `(source_id, external_id)` | Partial unique index |

### 2.3 Related tables

| Table | Role | Universal readiness |
|-------|------|---------------------|
| `organizations` | id, name, website_url | Minimal; no `organization_type` |
| `opportunity_sources` | Ingestion config per source/board | Employment-focused adapters |
| `employer_boards` | Registry metadata (migrations only) | Name implies employers only |
| `saved_opportunities` | Student bookmarks | **Type-agnostic** ✓ |
| `applications` | Tracker with interview pipeline | **Employment-biased** |
| `student_profiles` | Personalization input | Employment-biased fields |

### 2.4 Ingestion normalization

- `ingestrecord.RawOpportunity` is generic enough for non-employment **if adapters exist**
- `NormalizeCategory()` maps to existing category enum
- Relevance v2 `ShouldPersistSourceRecord` filters for **student technical employment**
- All ATS adapters set `Skills: []` — skills not extracted

### 2.5 API & frontend

- List filters: `category`, `work_arrangement`, `career_family`, `experience_level`
- Frontend browse filters: search, category, work arrangement only
- Recommended feed: hard-filtered to `high_confidence_technical` employment
- Types: `OpportunitySummary.category`, optional relevance fields — no `opportunity_type`

---

## 3. Opportunity taxonomy

### 3.1 Final canonical `opportunity_type` values (Model C — approved)

**Decision:** Use a **broad `opportunity_type`** plus **employment-only dimensions**. Do not store `internship` in both `opportunity_type` and `experience_level`.

| `opportunity_type` | Description |
|--------------------|-------------|
| `employment` | Paid work: internships, co-ops, apprenticeships, new-grad and early-career jobs |
| `research` | REU, URP, summer lab programs, undergraduate research placements |
| `scholarship` | Financial award programs |
| `fellowship` | Stipend-based fellowship (research or professional development) |
| `program` | Immersion, insight, bridge, leadership development, corporate scholar programs |
| `event` | Conferences, hackathons, summits with attendance/registration |
| `competition` | Case competitions, data challenges, innovation contests |
| `other` | Residual |

**Employment-only dimensions** (NULL for all non-employment types):

| Field | Values | Purpose |
|-------|--------|---------|
| `experience_level` | `internship`, `co_op`, `apprenticeship`, `new_grad`, `early_career`, `unknown` | Candidate seniority band |
| `employment_mode` | `full_time`, `part_time`, `seasonal` (optional) | Schedule / employment mode |
| `work_arrangement` | `remote`, `hybrid`, `on_site` | Already exists |
| `career_family` | Existing v2 enum | Technical classification for employment browse/recommend |

**Remove from `experience_level`:** `fellowship` — fellowships are an `opportunity_type`, not an employment seniority band.

### 3.2 Relationship: `opportunity_type` vs `experience_level` vs `category`

**Problem today:** `category` mixes opportunity kinds (`scholarship`) with employment modes (`full_time`, `part_time`). `experience_level` incorrectly includes `fellowship` alongside `internship`.

**Final model:**

```
opportunity_type   = WHAT the opportunity is (employment, research, scholarship, ...)
experience_level   = WHO it targets — ONLY when opportunity_type = employment
employment_mode    = full_time | part_time | seasonal — ONLY when opportunity_type = employment
```

**Migration mapping from current `category`:**

| Current `category` | → `opportunity_type` | → `experience_level` | → `employment_mode` |
|--------------------|----------------------|----------------------|---------------------|
| internship | employment | internship | NULL |
| full_time | employment | new_grad or early_career | full_time |
| part_time | employment | new_grad or early_career | part_time |
| apprenticeship | employment | apprenticeship | NULL |
| fellowship | fellowship | NULL | NULL |
| scholarship | scholarship | NULL | NULL |
| research | research | NULL | NULL |
| hackathon | event | NULL | NULL (+ `type_metadata.event_subtype=hackathon`) |
| conference | event | NULL | NULL (+ `type_metadata.event_subtype=conference`) |
| leadership_program | program | NULL | NULL (+ `type_metadata.program_format=leadership`) |
| other | other | NULL | NULL |

**Rule:** `experience_level` MUST be NULL unless `opportunity_type = employment`. Never duplicate `internship` across both fields.

---

## 4. Universal core fields

| Field | Classification | Notes |
|-------|----------------|-------|
| id | CORE | |
| opportunity_type | CORE | New canonical dimension (or normalized category) |
| title | CORE | |
| organization_id / organization_name | CORE | Prefer FK over time |
| description | CORE | |
| source_id, external_id | CORE | Per-source identity |
| source_url, application_url | CORE | Official links |
| verification_status, verification_method | CORE | Trust |
| first_seen_at, last_seen_at, last_checked_at | CORE | Freshness |
| opens_at, deadline, starts_at, ends_at | CORE | All nullable; never invent |
| location | CORE | |
| location_type | TYPE-SPECIFIC | remote/in_person/hybrid — for events too |
| eligibility | CORE | Summary text |
| education_level | CORE | |
| fields_of_study | CORE | TEXT[] or JSONB |
| skills | DERIVED | Optional; employment-heavy today |
| tags | CORE | Lightweight cross-type labels |
| status (open/closed) | CORE | |
| type_metadata | TYPE-SPECIFIC | JSONB (see §5) |
| relevance_tier | DERIVED | Product policy per type |
| career_family | DERIVED | Employment browse/recommend |
| compensation | TYPE-SPECIFIC | Stipend/salary text |
| work_arrangement | TYPE-SPECIFIC | Employment |
| experience_level | TYPE-SPECIFIC | Employment |
| missed_sync_count | DERIVED | Sync sources only |

**UNNECESSARY for v1 universal expansion:** separate tables per type, embeddings, AI summaries.

---

## 5. Type-specific metadata strategy

### 5.1 Options compared

| Approach | Queryability | Validation | Migrations | API complexity | Recommendation fit | Verdict |
|----------|--------------|------------|------------|----------------|-------------------|---------|
| A. Wide `opportunities` table | High | DB CHECK explosion | Painful | Simple reads | Poor (sparse columns) | **Reject** |
| B. Core + type tables | High | Strong per type | Many migrations | Join complexity | Good | **Defer** until query proof |
| C. Core + JSONB `type_metadata` | Medium | App-layer schemas | Low | Medium | Good | **Recommended v1** |
| D. Hybrid (JSONB + promoted columns) | High for hot fields | Mixed | Moderate | Medium | Good | **Recommended v2** |

### 5.2 Recommended: Hybrid-lite (Phase 1 = JSONB)

**Phase 1:** Add `type_metadata JSONB` with per-type JSON schemas validated in Go.

Example schemas (documented, not implemented):

```json
// scholarship
{ "award_amount": "up to $5,000", "renewable": false, "financial_need_required": true }

// research / REU
{ "research_area": "computational biology", "stipend": "$6,000", "housing_provided": true, "citizenship_required": "US citizen or permanent resident" }

// conference
{ "event_start": "2026-10-15", "event_end": "2026-10-18", "registration_cost": "$150", "travel_support": true }

// competition
{ "team_size_max": 4, "prize": "$10,000", "submission_deadline": "2026-04-01" }
```

**Phase 2 (promote when measured):** Index hot JSONB paths (e.g. `research_area`, `award_amount`) or promote to columns.

### 5.3 Employment-specific (remain on core or JSONB)

- `work_arrangement`, `experience_level`, `compensation`, `employment_mode`

---

## 6. Organization model

### Current state

```sql
organizations (id, name, website_url, timestamps)
```

- Used in dev seed only; ingestion writes `organization_name` string
- No distinction between company, university, foundation, government

### Recommendation

**Minimal additive change** (when expanding sources):

| Field | Purpose |
|-------|---------|
| `organization_type` | company, university, government, nonprofit, foundation, professional_organization, research_lab, other |
| `aliases` | TEXT[] for dedup/display normalization |

**Do not redesign** unless cross-source dedup or publisher trust UI requires it. `organization_name` string remains acceptable for v1 curation.

---

## 7. Source strategy taxonomy

Separate **publisher** from **acquisition**:

| Dimension | Question |
|-----------|----------|
| Publisher | Who published this opportunity? (NASA, TMCF, Stripe) |
| Acquisition | How did CareerOS obtain it? |

### Recommended `acquisition_strategy` values

| Strategy | Meaning |
|----------|---------|
| `PUBLIC_API` | Documented public API (USAJobs, GH, NSF Award API) |
| `OFFICIAL_PUBLIC_FEED` | Employer-published public feed (ATS boards) |
| `AUTHORIZED_FEED` | Partner-provided feed with permission |
| `PARTNERSHIP` | Contractual data sharing |
| `OFFICIAL_PUBLIC_PAGE` | Structured/manual extraction from official HTML listing |
| `MANUAL_VERIFIED` | CareerOS staff verified official URL |
| `CAREER_CENTER_SUBMISSION` | Institution-submitted, reviewed |
| `COMMUNITY_SUBMISSION` | Student/employer URL submission, reviewed |
| `OFFICIAL_LINK_DISCOVERY` | Catalog entry linking out; metadata minimal |

### Student-facing provenance (target UI copy)

> **Source:** National Science Foundation  
> **Verification:** Official public source  
> **Last verified:** 3 hours ago

Not: "Guaranteed open."

---

## 8. Source research matrix

See [`scripts/opportunity_source_research.json`](../scripts/opportunity_source_research.json) for structured entries.

### Summary table

| Source | Categories | Access status | Integration strategy |
|--------|------------|---------------|---------------------|
| USAJobs | Employment, internships | PUBLIC_API_AVAILABLE | ✓ Integrated |
| Greenhouse / Lever / Ashby | Employment | PUBLIC_API_AVAILABLE | ✓ Integrated |
| NSF REU Directory | Research | OFFICIAL_LINK_DISCOVERY_ONLY | Manual curation + official links |
| NSF Award API | Research metadata | PUBLIC_API_AVAILABLE | Supplemental; not student app feed |
| TMCF | Scholarships, programs | MANUAL_VERIFIED_CURATION | Partner feed or manual |
| UNCF | Scholarships, programs | MANUAL_VERIFIED_CURATION | Manual from scholarships.uncf.org |
| NASA OSTEM / Pathways | Internships | OFFICIAL_LINK_DISCOVERY / USAJobs | Link-out; Pathways via USAJobs |
| Handshake EDU API | Jobs, events | AUTHORIZED_ACCESS_REQUIRED | Grambling partnership only |
| NSBE | Scholarships, jobs | MANUAL_VERIFIED_CURATION | smapply + career center partnership |
| LinkedIn | Jobs | NOT_CURRENTLY_INTEGRATABLE | Partner posting APIs only; no search ingestion |
| Indeed | Jobs | AUTHORIZED_ACCESS_REQUIRED | ATS partner Job Sync; not consumer search API |
| Jobright | Jobs | NOT_CURRENTLY_INTEGRATABLE | No public API |
| Simplify | Jobs | RESEARCH_NEEDED | No authoritative API found |

---

## 9. Aggregator policy

CareerOS distinguishes:

| Statement | Meaning |
|-----------|---------|
| "We can view this in a browser" | **Not sufficient** for ingestion |
| "We are authorized to store/redistribute" | **Required** for ingestion |

### Platform policy

| Platform | Legitimate mechanism | CareerOS stance |
|----------|---------------------|-----------------|
| LinkedIn | Talent Solutions partner posting APIs (restricted) | **NOT NOW** — no job search ingestion |
| Indeed | Job Sync / ATS partner agreements | **NOT NOW** — partner-only; Publisher API deprecated |
| Handshake | EDU API for institution partners | **Partnership path** via Grambling career center |
| Jobright | No public API | **NOT NOW** — no scraping |
| Simplify | RESEARCH_NEEDED | **NOT NOW** until official access verified |
| USAJobs / ATS boards | Public APIs/feeds | **Continue** — already authorized pattern |

### Prohibited (explicit)

- CAPTCHA bypass, authenticated scraping, rate-limit evasion, proxy rotation against ToS
- Undocumented private/mobile APIs
- Copying another aggregator's database
- Marketing scraped data as "verified"

---

## 10. Research-opportunity strategy

### Legitimate discovery paths

1. **NSF REU Site Search** — https://www.nsf.gov/funding/initiatives/reu/search  
   Official directory; students apply per-site. CareerOS should **curate links**, not invent deadlines.

2. **University research program pages** — Official public pages per institution (manual verified curation).

3. **NSF Award API** — Award metadata; useful for enrichment, **not** a direct student application catalog.

4. **NASA STEM programs** — Link to NASA STEM Gateway; Pathways via USAJobs.

### Important metadata for research type

| Field | Priority |
|-------|----------|
| research_area / discipline | High |
| institution | High |
| education_level | High |
| deadline | High (when published) |
| stipend, housing, travel | Medium |
| citizenship/residency | High when stated |
| duration, program dates | Medium |
| faculty/lab | Low (often unavailable) |

### Relevance for research (future)

- Gate on `opportunity_type=research`
- Use `education_level`, `fields_of_study`, skills overlap
- Do **not** apply `career_family` employment classifier as hard filter

---

## 11. HBCU sourcing strategy

### Differentiation goal

Surface opportunities especially valuable to HBCU students: TMCF/UNCF scholarships, corporate scholar programs, HBCU-focused fellowships, conferences (NSBE, etc.), leadership programs.

### Strategy tiers

| Tier | Mechanism | Examples | Pilot? |
|------|-----------|----------|--------|
| AUTOMATED | Public API/feed | USAJobs, ATS boards (existing) | ✓ Now |
| PARTNERSHIP | Institutional agreement | Handshake (Grambling), TMCF data share | Post-pilot |
| MANUAL VERIFIED | Staff curates official URLs | TMCF open scholarships, UNCF portal listings | **First expansion** |
| CAREER CENTER SUBMISSION | GSU career center submits | Grambling internships/scholarships | Pilot-friendly |
| STUDENT SUBMISSION | Reviewed URL submissions | §12 | Post-pilot |

### Realistic pilot sequence

1. **Grambling career center manual curation** (local trust, low legal risk)
2. **TMCF/UNCF scholarship listings** (manual verified from official portals)
3. **NSBE scholarships** (seasonal manual curation)
4. Partnership conversations with TMCF/UNCF for structured feeds

---

## 12. Community submission design (future)

**Not implemented in this milestone.**

### Workflow

```
URL submitted → validate URL → identify domain → duplicate check
→ classify opportunity_type → extraction policy check
→ verification queue → publish or reject
```

### States

`submitted` → `verification_pending` → `verified` | `rejected` | `duplicate` | `expired`

### Abuse prevention

- Rate limits per user
- Domain allowlist/blocklist
- No auto-publish
- Provenance badge: "Community submitted — verified by CareerOS"
- Duplicate detection on canonical application URL (§13)

---

## 13. Deduplication strategy

### Current (preserved)

- Per-source: UNIQUE `(source_id, external_id)`
- Safe, proven, no cross-source merging

### Cross-source (future, conservative)

**Triggers for cross-source dedup:** Only when **high-confidence signals** align:

| Priority | Signal | Action |
|----------|--------|--------|
| 1 | Normalized `application_url` exact match | Suppress duplicate listing; keep best-verified record |
| 2 | `organization_name` + `external_id` (ATS ID) | Link as same role across boards |
| 3 | `organization` + normalized `title` + `deadline` + `location` | Flag for manual review only |
| 4 | Fuzzy title matching | **Do not automate** |

**Principle:** False merge (combining two distinct roles) is worse than duplicate display.

**When justified:** After manual curation + career center submissions introduce overlap with ATS ingestion.

---

## 14. Verification & trust model

### Current lifecycle

`verified` → (missed syncs) → `stale` → (deadline/expiry) → `closed`

Plus `unverified` for dev/manual entries.

### Proposed extension (additive)

| Field | Purpose |
|-------|---------|
| `verification_method` | official_api, official_feed, manual_review, partner_feed, community_submitted |
| `verified_at` | Last verification timestamp (may differ from `last_checked_at`) |
| `source_publisher_name` | Display publisher (may differ from acquisition source) |

### Display levels (student-facing)

| Level | Meaning |
|-------|---------|
| Official source verified | From documented public API/feed or official page |
| Partner verified | Institutional/partner feed |
| Manually verified | CareerOS reviewed official URL |
| Community submitted | User-submitted, reviewed |
| Stale / Closed | Existing behavior |

**Do not add redundant status dimensions** — extend `verification_status` semantics via `verification_method` + copy, not parallel enums.

---

## 15. Application / tracker implications

### Current `applications` model

Statuses: `saved`, `preparing`, `applied`, `oa_assessment`, `interview`, `final_round`, `offer`, `rejected`, `withdrawn`, `closed`

**Clearly employment-interview oriented.**

### Recommendation

**Keep `applications` employment-focused for now.**

| Opportunity type | Tracker behavior |
|------------------|------------------|
| Internship, new-grad, co-op | Current application tracker ✓ |
| Scholarship, fellowship | Future: "submitted" / "awarded" / "not selected" — or track as saved only |
| Conference | Registration tracking — low priority |
| Research REU | Application tracker mostly works (`applied`, `interview`, `offer`) |

**Valid outcome:** Generalize later via `student_opportunity_actions` only if non-employment tracking becomes a pilot requirement. For universal **discovery**, applications generalization is **not required**.

---

## 16. Recommendation implications

### Current signals (employment)

| Factor | Weight | Valid for |
|--------|--------|-----------|
| career_family | 30 | Employment |
| experience_level | 20 | Employment |
| skills_overlap | 20 | Employment, research |
| work_arrangement | 10 | Employment |
| location | 10 | All with location |
| freshness | 5 | All |
| deadline_urgency | 5 | All with deadline |

### Future type-aware scoring (document only)

| Type | Primary factors |
|------|-----------------|
| Internship / new-grad | career_family, experience, skills, work_arrangement |
| Scholarship | major match, education_level, eligibility, financial need flags, deadline |
| Research | research_area, skills, education_level, citizenship eligibility |
| Fellowship | field, education_level, deadline, stipend (if present) |
| Conference | career interests, location, event date |
| Hackathon / competition | skills, deadline, team constraints |

### Evolution path

1. Filter candidates by `opportunity_type` preference (profile extension)
2. Select weight profile per type
3. Keep single scorer with type-specific weight maps — **not** multiple recommender services yet

---

## 17. Frontend Explore information architecture

**Not implemented — wireframe for future.**

```
Explore
├── All                    (verified, mixed types)
├── Internships & Jobs     (internship, new_grad, co_op, apprenticeship)
├── Research               (research, REU)
├── Scholarships           (scholarship)
├── Fellowships            (fellowship)
├── Programs               (career_program, leadership_program)
├── Events                 (conference, hackathon, competition)
└── Saved                  (existing)

Universal filters:  search, deadline range, location, education level
Employment filters: work arrangement, experience level, career family
Scholarship filters:  award amount range, renewable, major/field
Research filters:     discipline, stipend, housing
Event filters:        event date range, registration cost
```

**Principle:** Default view stays simple (All + type tabs). Category-specific filters appear only inside a type tab.

Current `/opportunities` browse maps to "Internships & Jobs" tab. `/recommended` remains employment-focused until type-aware recommendations ship.

---

## 18. Migration strategy

### Constraints to preserve

- 188+ verified opportunities
- Saved opportunities, applications, recommendations, notifications
- Ingestion reliability, background jobs, source verification

### Recommended phases

| Phase | Scope | Risk |
|-------|-------|------|
| **A** | Add `opportunity_type` (mapped from `category`), `type_metadata JSONB`, `verification_method`, optional `organizations.organization_type` | Low — additive |
| **B** | Backfill employment records; keep `category` as deprecated alias | Low |
| **C** | First non-employment source via **manual verified curation** (no new adapter complexity) | Low |
| **D** | Type-aware browse tabs + filters (employment tab unchanged as default) | Medium |
| **E** | Type-aware recommendation weight profiles | Medium |
| **F** | Community submission workflow | Medium |
| **G** | Cross-source dedup (URL-based) | Medium — conservative |

**Production employment pilot:** Proceed on current schema. Phases A–B can run **in parallel with pilot** if needed for first scholarship listings.

---

## 19. Ranked first-expansion sources

Scored 1–5 on: student value, HBCU value, legitimacy, data quality, freshness, effort, maintenance, architecture fit, differentiation.

| Rank | Source | Rationale |
|------|--------|-----------|
| **1** | **Grambling / pilot career center manual curation** | Highest trust, local differentiation, no legal risk, validates universal model |
| **2** | **TMCF open scholarships (manual verified)** | High HBCU value; official portal; no API required for v1 |
| **3** | **UNCF scholarships (manual verified)** | Same pattern; broad HBCU relevance |
| **4** | **NSF REU directory (link + curated metadata)** | High STEM student value; official source; RESEARCH_NEEDED on bulk extraction |
| **5** | **NSBE scholarships (seasonal manual)** | Professional org value; smapply portal; membership eligibility complexity |

**Not first:** Handshake (requires partnership), NASA OSTEM (multiple application systems), NSF Award API alone (wrong abstraction).

---

## 20. NOT NOW

| Item | Why deferred |
|------|--------------|
| LinkedIn job search/scraping | Partner-only posting APIs; no authorized search ingestion |
| Indeed consumer scraping | Publisher API deprecated; partner ATS integrations only |
| Jobright / Simplify scraping | No public authorized API; aggregator dependency |
| Handshake without Grambling partnership | EDU API institution-scoped; not a public aggregator |
| Arbitrary web crawling | Legal/trust risk; unbounded maintenance |
| AI web agents / LLM extraction | Scope guardrail; trust risk |
| Fuzzy cross-source dedup | Risk of merging distinct opportunities |
| Full `applications` generalization | Unnecessary for discovery-first expansion |
| Separate ingestion microservices | Modular monolith sufficient |
| Kafka / Redis / Elasticsearch | No measured need |
| Wide opportunities table (30+ nullable cols) | Migration and API complexity |
| Workable / new ATS | Scope guardrail |
| Embeddings / multi-recommender system | Premature; type-weight maps sufficient |

---

## 21. Final architecture decision

### A. Does the current `opportunities` model need modification before production pilot?

**NO** for an **employment-focused** production pilot.

**YES (minimal, additive)** before the **first non-employment opportunity** appears in the product.

### B. Minimum modification needed

1. `opportunity_type` (or normalize `category` in place with mapping)
2. `type_metadata JSONB` for type-specific fields
3. `verification_method` on opportunities
4. Optional `organizations.organization_type`
5. `acquisition_strategy` on `opportunity_sources` (or JSONB config)

**Not required:** new tables per type, applications rename, recommendation rewrite.

### C. When?

| Change | Timing |
|--------|--------|
| Employment pilot | **Now** — no schema blockers |
| Minimal universal fields (Phase A) | **Before first non-employment listings** |
| Full Explore/recommendation by type | **After initial pilot** |

### D. First non-employment source

**Grambling State University career center manual verified curation**, followed by **TMCF scholarship listings**.

Evidence: no API dependency, highest pilot trust, strongest HBCU differentiation, validates universal model with lowest legal/ops risk.

### E. What NOT to integrate right now

LinkedIn, Indeed consumer APIs, Jobright, Simplify scraping, Handshake without partnership, unauthorized REU bulk scraping, arbitrary aggregators.

---

## 22. Opportunity Coverage Gap Analysis

**Date:** 2026-08-30  
**Methodology:** Authoritative/public sources only — official websites, developer documentation, impact reports, and open-source repository READMEs. No authenticated member scraping, Slack scraping, or reverse-engineering of private APIs.

### 22.1 ColorStack investigation

ColorStack is a 501(c)(3) nonprofit serving **Black and Latinx undergraduate Computer Science students** in the United States. As of its FY24 impact report, the organization reports **16,000+ members** across **1,000+ schools**, **50+ corporate partners**, and **200+ job opportunities shared monthly** via Slack.

#### What types of opportunities does ColorStack distribute?

| Type | Evidence | Notes |
|------|----------|-------|
| SWE internships | colorstack.org, FY24 impact report | Primary volume; partner-company recruiting |
| New-grad / full-time SWE | colorstack.org, Stacked Up Summit | Career fairs + resume book distribution |
| Corporate insight / immersion programs | Partner events, Stacked Up Summit | Often partner-specific (e.g., Microsoft-sponsored summit) |
| Workshops, webinars, coaching | colorstack.org/students | Career development, not always application-tracked |
| Partner programs & events | colorstack.org | "Apply to programs/events in collaboration with our partners" |
| Resume book submissions | Impact report, summit announcements | Distribution to partner recruiters — not a public job board |
| Scholarships / financial aid | FY24 impact report | Mentioned as member benefit; not the primary volume driver |
| Hackathons / competitions | Wikipedia, community events | Occasional; secondary to employment |
| Fellowships / REUs / research | **Not evidenced as primary** | ColorStack is employment- and community-centric, not a research directory |

ColorStack's value proposition is **curated employment access + community**, not universal opportunity taxonomy coverage.

#### Public vs member-only

| Surface | Access | Opportunity content |
|---------|--------|---------------------|
| colorstack.org (marketing) | Public | Describes benefits; no live opportunity listings |
| Slack workspace | **Member-only** | Primary channel for job sharing (~200+/month per impact report) |
| Oyster Member Profile (`app.colorstack.io`) | **Member-only** | Built-in opportunities board (announced Dec 2024); member directory, events, gamification |
| Stacked Up Summit / career fairs | Registered attendees (free, membership encouraged) | Live recruiting + resume book |
| ColorStack wiki (`wiki.colorstack.org`) | Public documentation | Career center guides; static "2023 Roles & Opportunities" link list (may be stale) |
| GitHub `colorstackorg/oyster` | Public code | Open-source platform; **not** a data API for third parties |

**Verdict:** Opportunity listings are **overwhelmingly member-only**. Public pages describe the model; they do not expose a searchable, redistributable opportunity catalog.

#### Does Oyster contain opportunity discovery functionality?

**Yes — for members.** ColorStack's **Oyster** is their in-house open-source community platform (GitHub: `colorstackorg/oyster`). It is **not** Oyster HR (`oysterhr.com`), which is an unrelated global employment platform with its own partner API.

Oyster (ColorStack) includes:

- **Member Profile** — member directory, events, gamification, and (per Dec 2024 announcement) a **built-in opportunities board**
- **Admin Dashboard** — internal workflows including application review
- **API app** — background jobs and **webhook integrations with external services** (member-scoped; no public developer portal)

The opportunities board feature is **product functionality inside a gated member app**, not a public feed.

#### Distribution mechanisms

| Mechanism | Role | Integratable by CareerOS? |
|-----------|------|---------------------------|
| Slack | High-volume job sharing, peer support | **No** — member-only; Slack ToS prohibits unauthorized scraping |
| Oyster opportunities board | Structured member job search | **No** — authenticated member UI only |
| Email / newsletters | Announcements | **No** — subscriber-only |
| Career fairs (2×/year) | Recruiter matching, resume book | **No** — event-based; partnership path only |
| Partner resume book | Bulk resume distribution to sponsors | **No** — not a listing feed |
| Public wiki / website | Static guides | **Manual reference only** — not a live feed |

#### API, feed, partnership, webhooks

| Question | Finding |
|----------|---------|
| Documented public API? | **No** — no third-party developer documentation found |
| Authorized data feed? | **No** — not published |
| Partner/integration access? | **Yes, for corporate sponsors** — partnerships@colorstack.org; employer-side recruiting, not student-app data export |
| Webhooks / exports? | Oyster has internal webhook integrations (open-source monorepo); **no external opportunity export documented** |
| Open-source code as API? | Oyster source is public for **member learning/contributions**; contributions restricted to ColorStack members; code ≠ data redistribution rights |

#### Classification

**`PARTNERSHIP_RECOMMENDED`**

CareerOS cannot legitimately ingest ColorStack opportunity listings today. Member access does **not** grant redistribution rights to a third-party platform.

#### Attribution / link-back (anticipated)

Without a published data license, assume:

- Listings remain **ColorStack member benefits** — link students to ColorStack membership and original employer application URLs
- Any partnership would likely require **explicit data-sharing agreement**, ColorStack branding, and apply-through-original-source (employer ATS) behavior
- Do **not** present ColorStack-curated roles as CareerOS-verified without partner `verification_method=partner_feed`

#### What CareerOS should ask ColorStack for

Contact: **hello@colorstack.org** / **partnerships@colorstack.org**

| Ask | Purpose |
|-----|---------|
| Authorized opportunity feed (JSON/CSV/API) | Structured ingestion with permission |
| Fields: title, company, application URL, deadline, location, employment type, eligibility (member-only flags), posted_at, expires_at | Normalization |
| Provenance metadata | `discovered_via=ColorStack`, `canonical_application_url` from employer |
| Deduplication guidance | Which listings are exclusive vs. reposted from public ATS |
| Attribution requirements | Logo, "via ColorStack", membership eligibility disclaimers |
| Refresh cadence + webhook on new posts | Freshness without scraping |
| HBCU / HSI student referral partnership | Mission alignment — CareerOS surfaces opportunities; students apply via ColorStack or employer |
| Pilot scope | CS internships/new-grad only first; expand to programs/events later |

---

### 22.2 Future ColorStack partnership integration (design only)

**Not implemented.** Conceptual flow if authorized feed were granted:

```
ColorStack authorized feed/API
  → CareerOS source adapter (acquisition_strategy = PARTNERSHIP)
  → normalize to opportunity_type + core fields
  → cross-source deduplicate (application_url → employer ATS)
  → classify opportunity_type (internship | new_grad_job | career_program)
  → set verification_method = partner_feed
  → set provenance: discovered_via = ColorStack, canonical_source = employer ATS when matched
  → CareerOS discovery/browse
  → student clicks through to original application (ColorStack or employer)
```

#### Provenance + deduplication model

Many ColorStack listings originate from employer ATS boards CareerOS already monitors (Greenhouse, Lever, Ashby). Preserve both discovery credit and canonical publisher:

| Field | Example |
|-------|---------|
| `organization_name` | Stripe |
| `application_url` | `https://boards.greenhouse.io/stripe/jobs/123` |
| `canonical_source_id` | greenhouse / stripe board |
| `canonical_external_id` | `123` |
| `discovered_via` | `ColorStack` |
| `acquisition_strategy` | `PARTNERSHIP` |
| `verification_method` | `partner_feed` |
| `type_metadata.discovery_notes` | `"Also promoted to ColorStack members"` |

**Dedup rules (conservative):**

1. **Exact `application_url` match** with existing ATS record → suppress duplicate listing; attach ColorStack as `discovered_via` alias on canonical record
2. **ColorStack-exclusive programs** (no public ATS URL) → retain as separate `career_program` or `internship` with `source_publisher_name = ColorStack`
3. **Fuzzy title match only** → flag for manual review; do not auto-merge

**Student-facing copy:**

> **Employer:** Stripe · **Found via:** ColorStack · **Apply:** Official company application

---

### 22.3 Coverage benchmarking matrix

Benchmarked against authoritative/public descriptions of each platform's scope. **CareerOS Today** reflects the current employment-centric ingestion runtime (USAJobs + ATS boards, relevance v2 employment gate, ~188 verified employment listings at time of architecture review).

| Opportunity Category | CareerOS Today | ColorStack | TMCF | UNCF | Handshake | Jobright | Simplify | LinkedIn | Research Directories |
|----------------------|----------------|------------|------|------|-----------|----------|----------|----------|---------------------|
| SWE internships | **Strong** | Strong (CS-focused, curated) | Moderate (via partners) | Low–moderate | Strong (institution-scoped) | Strong (aggregated) | Strong (aggregated) | Strong | Low |
| New-grad SWE | **Strong** | Strong | Moderate | Low | Strong | Strong | Strong | Strong | Low |
| Co-ops | Moderate | Low | Low | Low | Moderate | Moderate | Moderate | Moderate | Low |
| Federal tech internships | Moderate (USAJobs) | Low | Low | Low | Low | Low | Low | Low | Low |
| REUs / undergrad research | **None** | Low | Low | Low | Low | None | None | None | **Strong** (NSF, Pathways to Science) |
| Summer research programs | **None** | Low | Low | Low | Low | None | None | None | **Strong** |
| Research fellowships | **None** | Low | Low | Low–moderate | Low | None | None | Low | Moderate (NSF, ProFellow) |
| Scholarships | **None** (schema only) | Low (member aid mentioned) | **Strong** | **Strong** | Low | None | None | None | Moderate (Pathways to Science) |
| HBCU-specific programs | **None** | Moderate (BL CS focus) | **Strong** | **Strong** | Moderate (if partnered) | Low | Low | Low | Moderate |
| Corporate insight / immersion | **None** | Moderate (partner events) | Moderate (Leadership Institute) | Low–moderate | Moderate (events) | Low | Low | Low | Low (uni career center lists) |
| Leadership development | **None** | Moderate | **Strong** | Moderate | Moderate | None | None | Low | Low |
| Career fairs / conferences | **None** | **Strong** (Stacked Up Summit) | **Strong** | Moderate | **Strong** | None | None | Moderate | Moderate (NSBE, etc.) |
| Hackathons / competitions | **None** | Low–moderate | Low | Low | Moderate (events) | None | None | None | Moderate (Devpost) |
| Non-SWE tech (PM, design, data) | Weak | Moderate | Moderate | Low | Strong | Strong | Strong | Strong | Low |

**Gap severity legend:** Strong = category is a core product focus; Moderate = present but partial; Low = incidental; None = not in product scope today.

**Key insight:** CareerOS's largest gaps are **not** in SWE employment — aggregators (Simplify, Jobright, LinkedIn) and community platforms (ColorStack) largely overlap with ATS sources CareerOS already ingests. CareerOS's differentiation gap is **non-employment and HBCU-specific opportunity types** that aggregators also under-serve.

---

### 22.4 Research opportunity discovery sources

Goal: continuously discover open undergraduate research opportunities from authoritative sources.

| Source | Categories | Access classification | Repeatable? | Notes |
|--------|------------|----------------------|-------------|-------|
| [NSF REU Site Search](https://www.nsf.gov/funding/initiatives/reu/search) | REU, summer research | `PUBLIC_STRUCTURED_SOURCE` | Yes (seasonal refresh) | Official directory; students apply per-site; no central application API |
| [NSF Award Search API](https://api.nsf.gov/services/v1/awards.json) | Research awards, REU sites (metadata) | `API` | Yes | Supplemental enrichment; not a student application catalog; filter by REU program codes |
| [Pathways to Science](https://www.pathwaystoscience.org/) | REU, research internships, scholarships | `PUBLIC_STRUCTURED_SOURCE` | Yes (manual/periodic) | 1,000+ programs; no public API; IBP membership for program posting, not data export |
| [DOE SULI/CCI](https://science.osti.gov/wdts/suli) | National lab research internships | `OFFICIAL_PUBLIC_PAGE` | Yes (annual cycles) | Centralized application portal; no public listing API |
| [NASA STEM Gateway](https://www.nasa.gov/stem-content/nasa-stem-gateway/) | Internships, challenges, research | `OFFICIAL_PUBLIC_PAGE` | Yes | Salesforce-based; no public developer API; link-out discovery |
| [NASA Pathways](https://www.nasa.gov/careers/pathways/) | Federal internships | Partial via USAJobs | Yes | CareerOS already has USAJobs pattern |
| [NIH SIP](https://www.training.nih.gov/programs/sip) | Biomedical research internships | `OFFICIAL_PUBLIC_PAGE` | Yes (annual) | Official program page; manual curation |
| [Grants.gov search2 API](https://www.grants.gov/api/common/search2) | Federal grants (incl. research fellowships) | `API` | Yes | Public POST endpoint; requires filtering for student-relevant opportunities |
| [Simpler.Grants.gov API](https://api.simpler.grants.gov) | Federal funding opportunities | `API` | Yes | API key required; modern search/filter |
| University REU pages (per institution) | Site-specific REU | `MANUAL_VERIFIED` | Per-site | High maintenance; use NSF directory as index |
| [NIH NRSA / individual fellowships](https://www.nih.gov/) | Graduate fellowships | `OFFICIAL_PUBLIC_PAGE` | Yes | Less undergraduate-focused |

**Recommended research ingestion sequence:**

1. **NSF REU directory** — `PUBLIC_STRUCTURED_SOURCE` + official site links (`MANUAL_VERIFIED` metadata)
2. **NSF Award API** — enrich with award metadata where program element = REU
3. **DOE SULI/CCI** — annual manual verified entries with application portal links
4. **Pathways to Science** — partnership inquiry to IBP (ldetrick@ibparticipation.org) for authorized feed; until then, `MANUAL_VERIFIED` for high-value program categories
5. **Grants.gov API** — filtered pipeline for student-eligible STEM grants (lower precision; needs relevance gate)

---

### 22.5 Fellowship, scholarship, and program discovery sources

| Source | Categories | Access classification | Repeatable? | Notes |
|--------|------------|----------------------|-------------|-------|
| [TMCF Open Scholarships](https://tmcf.org/scholarships/open-scholarships/) | Scholarships, corporate scholar programs | `MANUAL_VERIFIED` → `PARTNERSHIP` | Yes (annual cycles) | partners@tmcf.org for talent sourcing; no public opportunity API |
| [UNCF Scholarships Portal](https://scholarships.uncf.org/) | Scholarships, internships, fellowships | `MANUAL_VERIFIED` → `PARTNERSHIP` | Yes (monthly additions) | Account-gated application; public program descriptions |
| [NSBE Scholarships](https://nsbe.org/scholarships/) | Scholarships | `MANUAL_VERIFIED` | Seasonal | smapply portal |
| [ProFellow](https://www.profellow.com/) | Fellowships, funded grad programs | `PUBLIC_STRUCTURED_SOURCE` | Yes (directory) | Free account; no API; link-out discovery |
| [Pathways to Science](https://www.pathwaystoscience.org/) | Fellowships, scholarships, research | `PUBLIC_STRUCTURED_SOURCE` | Yes | Same as §22.4 |
| Corporate program pages (e.g., [Google BOLD](https://www.google.com/), [Microsoft Explore](https://careers.microsoft.com/)) | Insight programs, immersion | `MANUAL_VERIFIED` | Annual | No central API; career center lists as coverage signal |
| [Code2040 Fellows](https://www.code2040.org/) | Tech fellowship pipeline | `OFFICIAL_PUBLIC_PAGE` | Annual | BL/Latinx focus; application on own site |
| [Devpost](https://devpost.com/) | Hackathons, competitions | `RESEARCH_NEEDED` | Yes | **No official API**; unofficial scrapers exist — not recommended for CareerOS |
| [MLH](https://mlh.io/) | Hackathons | `OFFICIAL_PUBLIC_PAGE` | Yes | Seasonal; manual curation |
| Uni career center lists (Tufts, etc.) | Insight programs, diversity programs | `MANUAL_VERIFIED` | Static reference | Good coverage signal; not automatable |

---

### 22.6 Source-of-source strategy (aggregator → authoritative)

For platforms CareerOS cannot ingest (ColorStack, Jobright, Simplify, LinkedIn, Handshake without partnership), use them as **coverage signals** — not data sources.

```
External platform listing observed (manual sample or partner feed)
  → extract organization name + role title
  → locate authoritative application URL (employer ATS, government portal, foundation site)
  → check if CareerOS already monitors that publisher
  → if yes: mark coverage OK (optionally note discovered_via)
  → if no: classify publisher as new source candidate
  → add publisher to opportunity_sources watchlist or manual curation queue
```

| Platform | Legitimate signal use | Prohibited |
|----------|----------------------|------------|
| ColorStack | Identify employers actively recruiting underrepresented CS talent; detect ColorStack-exclusive programs | Scraping member board / Slack |
| Simplify | GitHub internship lists (community OSS); employer list as watchlist for ATS boards | Scraping simplify.jobs (no public API per [llms.txt](https://simplify.jobs/llms.txt)) |
| Jobright | Employer/title trends for ATS board expansion | Any automated ingestion (no public API) |
| LinkedIn | Employer hiring activity signal | Job search scraping |
| Handshake | Institution-scoped via Grambling EDU API only | Cross-institution aggregation |
| Pathways to Science / ProFellow | Program publisher discovery | Bulk scraping behind login |

**Simplify note:** [simplify.jobs/llms.txt](https://simplify.jobs/llms.txt) explicitly states: *"Public API / OpenAPI spec / webhooks / MCP server: not currently published. For partnership or data-access inquiries, contact support@simplify.jobs."* Community GitHub lists (e.g., Summer2026-Internships) are **unofficial aggregations** — useful as board-expansion signals, not authoritative verification.

---

### 22.7 Answers to required questions

**1. What important opportunity categories is CareerOS currently missing?**

- REUs and undergraduate research programs
- Scholarships (especially HBCU-relevant)
- Fellowships (research and professional)
- Corporate insight / immersion / bridge programs
- HBCU-specific and diversity-focused programs (TMCF, UNCF, NSBE, ColorStack-exclusive)
- Conferences, career fairs (as discoverable events)
- Hackathons and competitions

**2. Which authoritative sources could fill each gap?**

| Gap | Primary sources |
|-----|-----------------|
| REUs / research | NSF REU directory, Pathways to Science, DOE SULI, NASA STEM Gateway |
| Scholarships | TMCF, UNCF, NSBE |
| Fellowships | ProFellow, Pathways to Science, Grants.gov |
| Corporate programs | Employer official pages + career center curated lists |
| HBCU programs | TMCF, UNCF, Grambling career center |
| Hackathons | Devpost/MLH (manual curation; no official API) |
| Employment (incremental) | Expand ATS employer board registry (already the core strategy) |

**3. Which can be automated legitimately?**

| Automatable now | Approach |
|-----------------|----------|
| SWE employment | ATS APIs + USAJobs (already integrated) |
| Federal grants/fellowships | Grants.gov `search2` API (with student relevance filter) |
| REU metadata enrichment | NSF Award API |
| NASA Pathways/federal internships | USAJobs (partial) |

| Automatable with structured public pages (no API) | Approach |
|---------------------------------------------------|----------|
| NSF REU directory | Scheduled link verification + curated metadata |
| DOE SULI/CCI cycles | Annual template records with portal links |

**4. Which require partnership?**

| Source | Why |
|--------|-----|
| **ColorStack** | Member-only distribution; no public feed |
| **Handshake** | EDU API institution-scoped (Grambling path) |
| **TMCF / UNCF** | High-value HBCU listings; partner outreach for structured feed |
| **Pathways to Science / IBP** | No public API; 1,000+ program directory |
| **Simplify** | No public API; partnership inquiry only |
| **LinkedIn / Indeed** | Partner-only posting APIs |
| **Jobright** | No authorized access |

**5. Which require manual verification?**

- Grambling career center submissions (pilot)
- TMCF/UNCF scholarship tiles (until partner feed)
- NSBE seasonal scholarships
- Corporate insight programs (per-employer annual cycles)
- University-specific REU pages linked from NSF directory
- Hackathon listings (Devpost — if curated without scraping)

**6. Where could ColorStack materially improve coverage?**

- **Diversity-focused SWE internships/new-grad** roles promoted to BL/Latinx CS students
- **ColorStack-exclusive partner programs** not on public ATS boards
- **Career fair / summit employer roster** as a source-of-source for board expansion
- **Community trust signal** — "employers actively recruiting through ColorStack" badge on canonical ATS records
- **Not** REUs, scholarships, or non-CS categories — outside ColorStack's core distribution

**7. Could CareerOS obtain ColorStack opportunities legitimately today?**

**No.** No documented public API, authorized feed, or export mechanism exists. Member access does not grant third-party redistribution rights.

**8. If not, what partnership/feed should CareerOS request?**

See §22.1 ask table: authorized JSON/CSV/API feed with application URLs, deadlines, eligibility, provenance fields, attribution requirements, refresh cadence, and dedup guidance — scoped initially to CS internships and new-grad roles.

**9. Which aggregator opportunities are already discoverable through employer sources?**

Significant overlap is expected between aggregator SWE internship and new-grad listings and employer ATS boards (Greenhouse, Lever, Ashby, Workday public pages) that CareerOS already ingests or can add by board token expansion — but this has **not been empirically measured** in CareerOS. Treat overlap as a hypothesis until a deduplication audit is run. Aggregator value is **curation, timing, and exclusives** — not necessarily unique employer data.

Detection method: match normalized `application_url` host against known ATS patterns (`boards.greenhouse.io`, `jobs.lever.co`, `jobs.ashbyhq.com`, etc.).

**10. Top 5 coverage gaps to address after the employment pilot**

| Rank | Gap | Recommended action | Access path |
|------|-----|-------------------|-------------|
| **1** | HBCU scholarships & programs | TMCF + UNCF manual verified curation | `MANUAL_VERIFIED` → partnership |
| **2** | REUs & summer research | NSF REU directory + official site links | `PUBLIC_STRUCTURED_SOURCE` |
| **3** | Grambling / career center local opportunities | Career center submission workflow | `CAREER_CENTER_SUBMISSION` |
| **4** | Corporate insight / bridge programs | Curated list from official employer pages | `MANUAL_VERIFIED` |
| **5** | Diversity employment exclusives | ColorStack partnership inquiry | `PARTNERSHIP` |

Employment ATS expansion remains important but is **lower differentiation** — aggregators already cover it. Post-pilot priority should shift to categories students cannot get from Simplify/LinkedIn alone.

---

## Appendix: Current vs target (conceptual)

```
TODAY                          TARGET (incremental)
─────                          ────────────────────
category (mixed enum)     →    opportunity_type (canonical)
experience_level (emp)    →    experience_level (employment only)
work_arrangement (emp)    →    employment fields / type_metadata
career_family (emp)       →    derived; employment browse only
relevance_tier (emp)      →    per-type product policy
organization_name       →    organization FK + type (gradual)
source_id/external_id   →    preserved per-source dedup
applications (interview)  →    employment tracker; defer generalization
```

---

*End of architecture document. No implementation performed.*
