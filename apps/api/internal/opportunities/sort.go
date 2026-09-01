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

func employmentSortSQL(prefix string, sort string) string {
	firstSeen := columnRef(prefix, "first_seen_at")
	created := columnRef(prefix, "created_at")
	deadline := columnRef(prefix, "deadline")
	arrangement := columnRef(prefix, "work_arrangement")

	switch sort {
	case SortDeadline:
		return deadline + " ASC NULLS LAST, " + firstSeen + " DESC, " + created + " DESC"
	case SortArrangement:
		return workArrangementRankSQL(arrangement) + ", " + firstSeen + " DESC, " + created + " DESC"
	default:
		return firstSeen + " DESC, " + created + " DESC"
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
	firstSeen := columnRef(prefix, "first_seen_at")
	created := columnRef(prefix, "created_at")
	deadline := columnRef(prefix, "deadline")
	arrangement := columnRef(prefix, "work_arrangement")

	switch sort {
	case SortDeadline:
		return fmt.Sprintf(`CASE
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'open' THEN 1
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN %s = 'employment' THEN 3
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END, %s ASC NULLS LAST, %s DESC, %s DESC`,
			oppType, meta, oppType, meta, oppType, oppType, meta, oppType, meta,
			deadline, firstSeen, created)
	case SortArrangement:
		return fmt.Sprintf(`CASE
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'open' THEN 1
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN %s = 'employment' THEN 3
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END, %s, %s DESC, %s DESC`,
			oppType, meta, oppType, meta, oppType, oppType, meta, oppType, meta,
			workArrangementRankSQL(arrangement), firstSeen, created)
	default:
		return fmt.Sprintf(`CASE
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'open' THEN 1
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'upcoming' THEN 2
			WHEN %s = 'employment' THEN 3
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'unknown' THEN 4
			WHEN %s = 'research' AND COALESCE(%s->>'application_status', 'unknown') = 'closed' THEN 5
			ELSE 6
		END, %s DESC, %s DESC, %s`,
			oppType, meta, oppType, meta, oppType, oppType, meta, oppType, meta,
			firstSeen, created, workArrangementRankSQL(arrangement))
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
		END, deadline ASC NULLS LAST, first_seen_at DESC, created_at DESC`
	case CatalogScopeAll:
		return mixedCatalogOrderBy("", filter.Sort)
	default:
		return employmentSortSQL("", filter.Sort)
	}
}

func dedupPickOrderSQL() string {
	return `(so.id IS NOT NULL) DESC, o.first_seen_at DESC, o.last_checked_at DESC NULLS LAST, o.created_at DESC`
}
