package ingestion

import (
	"github.com/careeros/api/internal/ingestion/ingestrecord"
	v2 "github.com/careeros/api/internal/ingestion/relevance/v2"
)

// ApplyClassification enriches a raw opportunity with Relevance Engine v2 fields.
func ApplyClassification(raw *ingestrecord.RawOpportunity) v2.Classification {
	return raw.ApplyClassification()
}

// ShouldRetainPosting applies the v2 ingest gate for adapter filtering.
func ShouldRetainPosting(title, description string) (bool, string, v2.Classification) {
	c, ok, reason := v2.EvaluateIngest(title, description)
	return ok, reason, c
}
