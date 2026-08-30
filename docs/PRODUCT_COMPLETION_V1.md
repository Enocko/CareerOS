# CareerOS Product Completion v1

This document summarizes the final student-facing product, trust model, operator tooling, and known limitations before production deployment.

## 1. Final student experience

Students register, complete a profile, and discover opportunities through:

- **For You** — deterministic personalized recommendations with match score labels (not calibrated percentages)
- **Browse** — unified catalog with All | Internships & Jobs | Research tabs
- **Saved** — bookmarked opportunities with closed/stale indicators and unsave
- **Applications** — employment application tracker with status history
- **Notifications** — deadline reminders linked to opportunities
- **Opportunity detail** — source freshness, research availability semantics, report-an-issue action

## 2. Opportunity trust model

Two layers:

| Layer | Field | Meaning |
|-------|-------|---------|
| Source trust | `verification_status` | verified / stale / unverified / closed |
| Research apply-now | `type_metadata.application_status` | open / upcoming / closed / unknown |

NSF award verification ≠ applications open. Employment listings use source sync freshness; research uses separate availability verification workflow.

## 3. Freshness lifecycle (employment)

```
discovered → verified → rechecked each successful sync
  → missing from successful exhaustive sync → missed_sync_count++
  → after threshold (2) → stale
  → past deadline while absent from sync → closed (status + verification_status)
  → authoritative reappearance at same (source_id, external_id) → reopen (stable ID)
```

**Invariants:**

- Failed ingestion runs do **not** call `ApplyPostSyncActions`. Provider outage cannot close listings.
- **First** unexpected exhaustive empty sync (`raw_fetched = 0`, `retained = 0`, `Exhaustive = true`) while the source has verified open listings records `empty_sync_anomaly` and leaves listings unchanged.
- **Second consecutive** exhaustive empty sync (immediately prior run was `empty_sync_anomaly` with `raw=0`/`retained=0`) confirms authoritative emptiness; post-sync absence processing may proceed.
- After a successful authoritative-empty sync (`raw=0`, `retained=0`), subsequent exhaustive empty syncs continue post-sync without re-confirmation.
- **`raw_fetched > 0` but `retained = 0`:** provider returned postings but none are CareerOS-relevant — post-sync is skipped (success, listings unchanged). This does **not** prove prior external IDs disappeared.
- **`Exhaustive = false`:** incomplete pagination or unvalidated response — run fails (`incomplete_sync`), listings unchanged.
- Listings **confirmed present** in the current sync are not deadline-closed in that same run — authoritative source presence wins over an expired deadline field.

### Empty-sync decision table

| Condition | Post-sync | Run outcome |
|-----------|-----------|-------------|
| `retained > 0` | Apply (normal) | Success |
| `verified_open = 0` | Apply (noop) | Success |
| `raw > 0`, `retained = 0` | Skip | Success |
| `raw = 0`, `!Exhaustive` | Skip | Fail (`incomplete_sync`) |
| `raw = 0`, first exhaustive empty | Skip | Fail (`empty_sync_anomaly`) |
| `raw = 0`, 2nd consecutive exhaustive empty | Apply | Success |
| `raw = 0`, ongoing established empty board | Apply | Success |

Constants: `EmptySyncGuardMinVerifiedOpen = 1`, `EmptySyncConfirmationsRequired = 2`.

Adapters set `Exhaustive` and `AuthoritativeEmpty` on successful complete fetches via `FetchResult.MarkExhaustiveSuccess()`.

## 3a. Closed listing access

Browse excludes closed, stale, and test fixtures. Detail access rules:

| Access path | Closed/stale reachable? |
|-------------|-------------------------|
| Browse discovery | No |
| Direct URL without prior relationship | No (404) |
| Saved | Yes |
| Application tracked | Yes |
| Prior detail view (`opportunity_views`) | Yes |

Detail pages show **“This opportunity is closed”** and disable new applications / apply actions. Existing saves, applications, and notes are preserved.

## 3b. Authoritative reopening

When the same `(source_id, external_id)` reappears in a successful sync after closure:

- `status` → `open`, `verification_status` → `verified`
- `last_seen_at` / `last_checked_at` updated, `missed_sync_count` reset
- Opportunity ID, saves, and application history unchanged
- No duplicate row created

Reopening requires exact source identity — not failed syncs, fuzzy title matches, or different postings.

## 4. Source health

Operators can inspect:

- `make catalog-report` — CLI catalog health summary
- `GET /api/v1/admin/overview` — API snapshot with per-source last run, open counts, consecutive failures
- `make jobs-report` — background job queue health

## 5. Reporting workflow

Students (authenticated) submit reports on opportunity detail pages:

```
student report → opportunity_reports (pending)
  → operator review (/admin)
  → mark resolved / dismissed
  → source re-check determines final listing status
```

Reports never auto-close listings.

## 6. Duplicate strategy

Cross-source dedup is **diagnostic only** (`make catalog-report`). Deterministic grouping by normalized organization + title. No fuzzy/LLM dedup in v1. If duplicate rate is material post-deployment, consider conservative display suppression.

## 7. Search

PostgreSQL `ILIKE` across title, organization, description, research area, cycle label, skills, tags. Scoped by Browse tab (`type` param). No Elasticsearch.

## 8. Browse

- Nav: **Browse** → page header **Opportunities**
- All feed ordering prioritizes actionable research (open → upcoming) then employment, then unknown/closed research
- Closed and stale employment excluded from browse; test fixtures excluded by external_id prefix

## 9. For You

Deterministic scorer using profile interests, roles, skills, location, experience. Match display: `Strong match · 91` (score is relative, not statistically calibrated).

## 10. Saved

Shows all saved items including closed. Displays listing status warnings. Unsave in place.

## 11. Applications

Employment only. Duplicate applications blocked. Closed opportunities rejected at create.

## 12. Research

Unified in Browse. Unknown availability shows program website, not Apply. Admin verification at `/admin/research`.

## 13. Notifications

In-app only. Unread count, mark read, mark all read. No email/SMS in v1.

## 14. Operator tooling

| Tool | Purpose |
|------|---------|
| `/admin` | Overview, source health, pending reports |
| `/admin/research` | REU availability verification queue |
| `make catalog-report` | Full catalog metrics |
| `make jobs-report` | Job queue metrics |

## 15. Catalog metrics

Run `make catalog-report` for employment/research counts, provider breakdown, duplicate audit, URL sample audit, source health.

## 16. Remaining limitations

- No new ATS providers (Workday, SmartRecruiters, etc.)
- No scholarship catalog or generalized non-employment tracker
- No AI/embeddings recommendations
- Duplicate suppression not automated (~22% diagnostic duplicate candidate rate)
- HTTP URL audit is heuristic only (no browser automation)
- Dashboard is the Applications page (no separate vanity dashboard)
- Empty-sync guard uses documented thresholds (`EmptySyncGuardMinVerifiedOpen = 1`, `EmptySyncConfirmationsRequired = 2`); no large-drop ratio guard in v1
- Cross-source dedup remains diagnostic only

## 17. Product Completion v1 checklist (final)

| # | Area | Status |
|---|------|--------|
| 1 | Freshness lifecycle | Complete |
| 2 | Failed-sync protection | Complete |
| 3 | Empty-sync anomaly guard | Complete |
| 4 | Authoritative reopening | Complete |
| 5 | Closed detail access (saved/tracked/viewed) | Complete |
| 6 | Browse trust filters | Complete |
| 7 | Employment URL validation | Complete |
| 8 | Opportunity reporting | Complete |
| 9 | Admin overview + research verification | Complete |
| 10 | Catalog report CLI | Complete |
| 11 | Unified Browse | Complete |
| 12 | Research sections in Browse | Complete |
| 13 | For You recommendations | Complete |
| 14 | Saved with closed/stale badges | Complete |
| 15 | Application tracker | Complete |
| 16 | Profile completeness | Complete |
| 17 | Notifications | Complete |
| 18 | Detail freshness + report issue | Complete |
| 19 | Test data isolation | Complete |
| 20 | Integration test coverage | Complete |

**Conclusion:** `CAREEROS PRODUCT V1 COMPLETE` — ready for student use in development/staging; production deployment is a separate phase (not started).

## 18. Post-v1 roadmap

See catalog-report source coverage section and partnership-required sources:

- Handshake / Grambling relationship
- TMCF, UNCF, ColorStack partnerships
- Workday / custom career sites
- DOE / NASA research programs
- Email/SMS notifications
- Deterministic dedup display if duplicate rate warrants
