package opportunities

// SQL helpers for collapsing multi-location employment postings with the same
// normalized organization + title (e.g. Lever boards listing one role per city).

func normOrgSQL(column string) string {
	return "lower(trim(regexp_replace(" + column + ", '[^a-zA-Z0-9]+', ' ', 'g')))"
}

func normTitleSQL(column string) string {
	return normOrgSQL(column)
}

func employmentDedupPartitionSQL() string {
	normOrg := normOrgSQL("o.organization_name")
	normTitle := normTitleSQL("o.title")
	return `CASE
		WHEN o.opportunity_type = 'employment'
			AND ` + normOrg + ` <> ''
			AND ` + normTitle + ` <> ''
		THEN ` + normOrg + ` || E'\x1f' || ` + normTitle + `
		ELSE o.id::text
	END`
}

func shouldDedupeEmployment(scope string) bool {
	return scope == CatalogScopeAll || scope == CatalogScopeEmployment || scope == ""
}

func researchOrderBy() string {
	return `CASE COALESCE(o.type_metadata->>'application_status', 'unknown')
		WHEN 'open' THEN 1
		WHEN 'upcoming' THEN 2
		WHEN 'unknown' THEN 3
		WHEN 'closed' THEN 4
		ELSE 5
	END, o.deadline ASC NULLS LAST, o.created_at DESC`
}

func researchVisibilityCondition() string {
	return "o.opportunity_type = 'research'"
}
