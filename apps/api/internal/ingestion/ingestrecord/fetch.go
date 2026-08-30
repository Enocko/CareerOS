package ingestrecord

// FetchResult captures adapter-level ingestion metrics before persistence.
type FetchResult struct {
	RawFetched    int
	Retained      []RawOpportunity
	FilteredOut   int
	FilterReasons map[string]int
	// Exhaustive is true when the adapter completed a full source fetch: valid response
	// structure, successful parsing, and pagination finished (when applicable).
	Exhaustive bool
	// AuthoritativeEmpty is true when Exhaustive is true and the provider's complete board
	// or search result set contains zero postings (RawFetched == 0).
	AuthoritativeEmpty bool
}

// RetainedCount returns the number of postings kept after relevance filtering.
func (r FetchResult) RetainedCount() int {
	return len(r.Retained)
}

// MarkExhaustiveSuccess sets exhaustion metadata for a completed adapter fetch.
// AuthoritativeEmpty is set automatically when the exhaustive result has zero raw postings.
func (r FetchResult) MarkExhaustiveSuccess() FetchResult {
	r.Exhaustive = true
	if r.RawFetched == 0 && r.RetainedCount() == 0 {
		r.AuthoritativeEmpty = true
	}
	return r
}
