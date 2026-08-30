package relevance

// RecordFilterReason increments a filter-reason counter map.
func RecordFilterReason(reasons map[string]int, reason string) {
	if reasons == nil {
		return
	}
	if reason == "" {
		reason = ReasonUnknown
	}
	reasons[reason]++
}
