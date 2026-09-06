package opportunities

import "fmt"

func columnRef(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + name
}

func workArrangementRankSQL(column string) string {
	return fmt.Sprintf(`CASE %s
		WHEN 'remote' THEN 1
		WHEN 'hybrid' THEN 2
		WHEN 'flexible' THEN 3
		WHEN 'on_site' THEN 4
		ELSE 5
	END`, column)
}

// locationPreferenceSQL soft-ranks listings for HBCU / US student relevance.
// Remote and domestic roles surface ahead of overseas on-site postings.
func locationPreferenceSQL(prefix string) string {
	arrangement := columnRef(prefix, "work_arrangement")
	location := columnRef(prefix, "location")
	return fmt.Sprintf(`CASE
		WHEN %s = 'remote' THEN 1
		WHEN %s IN ('hybrid', 'flexible') THEN 2
		WHEN COALESCE(%s, '') = '' THEN 4
		WHEN %s ILIKE '%%india%%'
			OR %s ILIKE '%%hyderabad%%'
			OR %s ILIKE '%%bangalore%%'
			OR %s ILIKE '%%bengaluru%%'
			OR %s ILIKE '%%china%%'
			OR %s ILIKE '%%singapore%%'
			OR %s ILIKE '%%apac%%'
			OR %s ILIKE '%%emea%%'
		THEN 8
		ELSE 3
	END`, arrangement, arrangement, location,
		location, location, location, location, location, location, location, location)
}

func freshnessOrderSQL(prefix string) string {
	lastChecked := columnRef(prefix, "last_checked_at")
	firstSeen := columnRef(prefix, "first_seen_at")
	created := columnRef(prefix, "created_at")
	return lastChecked + " DESC NULLS LAST, " + firstSeen + " DESC, " + created + " DESC"
}

// newestOrderSQL prioritizes newly discovered roles, then recently verified ones.
// Location preference is only a tiebreaker so overseas stacks don't bury US/remote.
func newestOrderSQL(prefix string) string {
	firstSeen := columnRef(prefix, "first_seen_at")
	lastChecked := columnRef(prefix, "last_checked_at")
	created := columnRef(prefix, "created_at")
	return firstSeen + " DESC, " + lastChecked + " DESC NULLS LAST, " +
		locationPreferenceSQL(prefix) + ", " + created + " DESC"
}

func employmentSortSQL(prefix string, sort string) string {
	deadline := columnRef(prefix, "deadline")
	arrangement := columnRef(prefix, "work_arrangement")

	switch sort {
	case SortDeadline:
		return deadline + " ASC NULLS LAST, " + freshnessOrderSQL(prefix)
	case SortArrangement:
		return workArrangementRankSQL(arrangement) + ", " + freshnessOrderSQL(prefix)
	default:
		return newestOrderSQL(prefix)
	}
}

func plainOrderBy(filter ListFilter) string {
	switch filter.CatalogScope {
	case CatalogScopeResearch:
		return researchOrderBy()
	case CatalogScopeAll:
		return mixedCatalogOrderBy("o.", filter.Sort)
	default:
		return employmentSortSQL("o.", filter.Sort)
	}
}

func mixedCatalogOrderBy(prefix string, sort string) string {
	oppType := columnRef(prefix, "opportunity_type")
	meta := columnRef(prefix, "type_metadata")
	deadline := columnRef(prefix, "deadline")
	arrangement := columnRef(prefix, "work_arrangement")

	statusBucket := fmt.Sprintf(`CASE
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'open' THEN 1
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN %s = 'employment' THEN 3
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END`,
		oppType, meta, oppType, meta, oppType, oppType, meta, oppType, meta)

	switch sort {
	case SortDeadline:
		return statusBucket + ", " + deadline + " ASC NULLS LAST, " + freshnessOrderSQL(prefix)
	case SortArrangement:
		return statusBucket + ", " + workArrangementRankSQL(arrangement) + ", " + freshnessOrderSQL(prefix)
	default:
		return statusBucket + ", " + newestOrderSQL(prefix)
	}
}

func dedupedOrderBy(filter ListFilter) string {
	switch filter.CatalogScope {
	case CatalogScopeResearch:
		return `CASE COALESCE(type_metadata->>'application_status', 'unknown')
			WHEN 'open' THEN 1
			WHEN 'upcoming' THEN 2
			WHEN 'unknown' THEN 3
			WHEN 'closed' THEN 4
			ELSE 5
		END, deadline ASC NULLS LAST, last_checked_at DESC NULLS LAST, first_seen_at DESC, created_at DESC`
	case CatalogScopeAll:
		return mixedCatalogOrderBy("", filter.Sort)
	default:
		return employmentSortSQL("", filter.Sort)
	}
}

// usesOrgDiversity spreads employers within the same discovery day so one
// company cannot monopolize a single ingest batch, while newer days still
// always rank above older ones.
func usesOrgDiversity(filter ListFilter) bool {
	if filter.CatalogScope == CatalogScopeResearch {
		return false
	}
	switch filter.Sort {
	case SortDeadline, SortArrangement:
		return false
	default:
		return true
	}
}

// orgDiversityPartitionSQL groups by employer + UTC discovery day so today's
// new roles are not delayed behind yesterday's diversified page-1 fill.
func orgDiversityPartitionSQL() string {
	normOrg := normOrgSQL("organization_name")
	return `CASE
		WHEN opportunity_type = 'employment' AND ` + normOrg + ` <> ''
		THEN ` + normOrg + ` || E'\x1f' || ((first_seen_at AT TIME ZONE 'UTC')::date)::text
		ELSE id::text
	END`
}

// diversifiedFinalOrderSQL keeps brand-new discovery days on top, then
// round-robins employers within that day.
func diversifiedFinalOrderSQL(filter ListFilter) string {
	dayBucket := `(first_seen_at AT TIME ZONE 'UTC')::date DESC`
	withinDay := `org_slot ASC, ` + newestOrderSQL("")
	switch filter.CatalogScope {
	case CatalogScopeAll:
		// Preserve research application-status priority from mixed catalog sort.
		oppType := "opportunity_type"
		meta := "type_metadata"
		statusBucket := fmt.Sprintf(`CASE
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'open' THEN 1
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN %s = 'employment' THEN 3
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END`,
			oppType, meta, oppType, meta, oppType, oppType, meta, oppType, meta)
		return statusBucket + ", " + dayBucket + ", " + withinDay
	default:
		return dayBucket + ", " + withinDay
	}
}

func dedupPickOrderSQL() string {
	return `(so.id IS NOT NULL) DESC, ` + newestOrderSQL("o.")
}
