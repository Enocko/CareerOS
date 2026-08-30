package ingestion

import (
	"context"
	"encoding/json"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
)

// Adapter fetches opportunities from a single source.
type Adapter interface {
	Name() string
	FetchAll(ctx context.Context, config json.RawMessage) (ingestrecord.FetchResult, error)
}

// AdapterFactory creates an adapter instance.
type AdapterFactory func() Adapter
