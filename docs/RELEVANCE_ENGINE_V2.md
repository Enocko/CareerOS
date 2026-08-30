# Relevance Engine v2 — Validation Report

**Milestone:** CareerOS Relevance Engine v2  
**Date:** 2026-08-30  
**Scope:** Deterministic classification quality (no AI/LLM, no new ATS providers)

---

## 1. Architecture Changes

### Source truth vs product relevance

CareerOS now treats **source-backed storage** and **product feed inclusion** as separate concerns:

| Layer | Responsibility |
|-------|----------------|
| **Ingestion gate** (`ShouldPersistSourceRecord`) | Persist postings with a student/entry-level experience signal (intern, co-op, new grad, early career, fellowship, apprenticeship, or entry-level engineer titles). |
| **Classification** (`relevance/v2.Classify`) | Assign experience level, career family, education level, relevance tier, and reason codes. |
| **Product feed** (API default) | Return only `relevance_tier = high_confidence_technical` unless `include_ambiguous` or `include_non_technical` is set. |

### Schema (migration `000008_relevance_engine_v2`)

New `opportunities` columns:

- `experience_level` — internship, co_op, new_grad, early_career, apprenticeship, fellowship, unknown
- `career_family` — software_engineering, data_science, machine_learning_ai, cybersecurity, product_management_technical, cloud_infrastructure_devops, quantitative_technology, technical_research, other_technical, non_technical, unknown
- `education_level` — undergraduate, masters, phd, graduate_any, unspecified
- `relevance_tier` — high_confidence_technical, ambiguous, high_confidence_non_technical
- `classification_reasons` — explainability codes (e.g. `internship + software_engineering`)

### Ingestion architectural note

**Current behavior:** Greenhouse, Ashby, and Lever adapters still **discard upstream postings before PostgreSQL persistence** when they lack a student/entry-level experience signal. Non-technical student roles (e.g. Marketing Intern) **are persisted** with `high_confidence_non_technical` tier.

**Tradeoff:** Full source-truth archival (storing every board posting) would require adapter refactors to persist all raw jobs and move all filtering to post-upsert classification. That is a larger redesign than this milestone. The smallest safe evolution implemented here:

1. Broadened ingest gate to include entry-level engineer titles (Software Engineer I, Associate Software Engineer, etc.).
2. Persist non-technical student roles instead of only “vaguely technical” intern titles.
3. Store classification on every upsert; product filtering happens at API layer.

**Non-technical opportunities are not deleted** from PostgreSQL when excluded from the technical browse feed.

### API changes

`GET /api/v1/opportunities` supports:

| Query param | Effect |
|-------------|--------|
| `career_family` | Filter by career family |
| `experience_level` | Filter by experience level |
| `include_ambiguous=true` | Include `ambiguous` tier in results |
| `include_non_technical=true` | Include non-technical student roles |

Default browse: **technical feed only** (`high_confidence_technical`).

### Tooling

- `make relevance-report` — reclassifies verified catalog and runs false-negative audit
- Regression tests: `apps/api/internal/ingestion/relevance/v2/classifier_test.go`

---

## 2. Classification Taxonomy

### Experience level

`internship` · `co_op` · `new_grad` · `early_career` · `apprenticeship` · `fellowship` · `unknown`

### Career family

`software_engineering` · `data_science` · `machine_learning_ai` · `cybersecurity` · `product_management_technical` · `cloud_infrastructure_devops` · `quantitative_technology` · `technical_research` · `other_technical` · `non_technical` · `unknown`

### Education level

`undergraduate` · `masters` · `phd` · `graduate_any` · `unspecified`

### Relevance tier policy

| Tier | Behavior |
|------|----------|
| **HIGH CONFIDENCE TECHNICAL** | Included in primary technical student browse |
| **AMBIGUOUS / UNKNOWN** | Retained in catalog; excluded from default browse; available via `include_ambiguous=true` |
| **HIGH CONFIDENCE NON-TECHNICAL** | Retained in catalog; excluded from technical browse; available via `include_non_technical=true` |

PhD-only research internships receive `phd_research_internship` reason and are tiered **ambiguous** (not promoted as ordinary undergraduate SWE opportunities).

---

## 3. Existing Opportunities Evaluated

**188** verified open opportunities reclassified.

---

## 4. Technical Retained (primary feed)

**90** opportunities (`high_confidence_technical`)

---

## 5. Non-Technical Excluded from Technical Browse

**9** opportunities (`high_confidence_non_technical`)

Includes the three manually confirmed false positives:

- Marketing Intern
- SDR Intern
- Customer Experience Associate (New Grad)

---

## 6. Ambiguous

**89** opportunities (`ambiguous`)

These are retained internally for broader discovery but not auto-classified into SWE/data/security families in the primary feed.

---

## 7. Career-Family Distribution

| Career family | Count |
|---------------|------:|
| software_engineering | 73 |
| unknown | 86 |
| non_technical | 9 |
| machine_learning_ai | 5 |
| cloud_infrastructure_devops | 5 |
| technical_research | 5 |
| data_science | 2 |
| product_management_technical | 2 |
| cybersecurity | 1 |

---

## 8. Experience-Level Distribution

| Experience level | Count |
|------------------|------:|
| internship | 97 |
| unknown | 43 |
| new_grad | 38 |
| early_career | 9 |
| co_op | 1 |

---

## 9. Education-Level Findings

| Education level | Count |
|-----------------|------:|
| unspecified | 169 |
| phd | 7 |
| masters | 5 |
| undergraduate | 5 |
| graduate_any | 2 |

PhD signals detected in titles/descriptions are preserved as metadata. PhD research internships are not promoted to the primary undergraduate technical feed.

---

## 10. Before / After False Positives

### BEFORE (v1)

| Metric | Value |
|--------|------:|
| Verified catalog | 188 |
| Technical feed (de facto: all verified) | 188 |
| Known false positives in technical browse | 3+ (Marketing Intern, SDR Intern, Customer Experience Associate New Grad) |

### AFTER (v2)

| Metric | Value |
|--------|------:|
| Source-backed catalog (unchanged row count) | 188 |
| Technical feed | 90 |
| Ambiguous | 89 |
| Non-technical (excluded from default browse) | 9 |

Known false positives are now **`high_confidence_non_technical`** and excluded from default student browse.

---

## 11. Confirmed False Negatives (v2 improvements over v1)

Live board audit found **6** titles v1 filtered but v2 correctly recovers as persistable student/entry-level technical roles:

- Software Engineer I, Backend (Collections) — Affirm
- Software Engineer I, Fullstack (Servicing International) — Affirm
- Bank Compliance Technology Analyst — Affirm (tiered `other_technical`; review warranted)
- Revenue Technology Analyst — GitLab (tiered `other_technical`; review warranted)

**Intentionally still filtered** (not false negatives):

- Staff / Principal / Senior + Engineer I (experienced IC levels, not student roles)
- Software Engineer II+ (mid-level, not student/new-grad)

---

## 12. Regression Test Results

All tests in `relevance/v2/classifier_test.go` pass, including:

- **Should include:** Software Engineer Intern, ML Intern, Security Engineering Intern, Technical Product Management Intern, Software Engineer I, Engineering Co-op, etc.
- **Should exclude from technical feed:** Marketing Intern, SDR Intern, Sales Intern, Customer Experience Associate (New Grad), Recruiting Intern, Finance Intern, Product Marketing Intern
- **Ambiguous:** Operations Associate New Grad, Business Analyst Intern
- **Guards:** Staff/Principal/Senior Software Engineer I not treated as entry-level

Full API test suite: **PASS**

---

## 13. Manual Audit Results (sampled)

### Retained technical (30 sampled)

Predominantly Palantir, Databricks, and similar **Software Engineer Intern / New Grad** roles with `software_engineering` family. No Marketing/SDR/Customer Experience titles in the technical sample.

### Rejected non-technical (9 total in catalog)

All 9 non-technical rows are visible in audit; includes Marketing Intern, SDR Intern, Customer Experience Associate (New Grad), Partner Marketing Intern, Technical Recruiter Early Career.

### Ambiguous (30 sampled)

Mix of USAJobs civil-service titles (Student Trainee IT, Computer Scientist), Palantir Deployment Strategist Intern, Product Designer New Grad, and consulting intern roles without clear technical family match. These are **retained** but not in default browse.

**Observed residual risk:** `Technology Analyst` titles may be finance/compliance rather than SWE; tiered `other_technical` — flagged for future rule refinement, not bulk-included without review.

---

## 14. Ingestion / Relevance Architectural Concern

**Finding:** Adapters still drop non-student senior postings before persistence. This is acceptable for the current student catalog scope but limits future “full board” analytics and false-negative auditing to live API scans (as performed by `make relevance-report`) rather than SQL queries over archived filtered titles.

**Recommended next evolution (not in this milestone):** optional `source_postings` staging table or ingest-all/classify-later adapter mode when source coverage expansion requires full-board recall measurement.

---

## Files Changed (summary)

- `apps/api/internal/ingestion/relevance/v2/` — classifier, types, ingest helpers, tests
- `apps/api/migrations/000008_relevance_engine_v2.*.sql`
- `apps/api/internal/ingestion/{greenhouse,ashby,lever}/adapter.go` — v2 ingest gate
- `apps/api/internal/ingestion/repository.go` — classification persistence
- `apps/api/internal/opportunities/` — API models, filters, technical feed default
- `apps/api/cmd/relevance-report/` — backfill + audit CLI
- `Makefile` — `relevance-report` target
