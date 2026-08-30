package ingestion

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/careeros/api/internal/ingestion/ashby"
	"github.com/careeros/api/internal/ingestion/greenhouse"
	"github.com/careeros/api/internal/ingestion/lever"
	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/careeros/api/internal/ingestion/nsf_reu"
	"github.com/careeros/api/internal/ingestion/usajobs"
)

// HTTPDoer is satisfied by *http.Client for adapter HTTP injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Credentials holds runtime secrets for ingestion adapters.
type Credentials struct {
	USAJobsAPIKey      string
	USAJobsUserAgent   string
}

// NewAdapterRegistry builds adapter factories with runtime credentials.
func NewAdapterRegistry(creds Credentials, httpClient HTTPDoer) map[string]AdapterFactory {
	return map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &credentialAdapter{
				inner:     usajobs.NewAdapter(httpClient),
				apiKey:    creds.USAJobsAPIKey,
				userAgent: creds.USAJobsUserAgent,
			}
		},
		"greenhouse": func() Adapter {
			return greenhouse.NewAdapter(httpClient)
		},
		"ashby": func() Adapter {
			return ashby.NewAdapter(httpClient)
		},
		"lever": func() Adapter {
			return lever.NewAdapter(httpClient)
		},
		"nsf_reu": func() Adapter {
			return nsf_reu.NewAdapter(httpClient)
		},
	}
}

type credentialAdapter struct {
	inner     *usajobs.Adapter
	apiKey    string
	userAgent string
}

func (a *credentialAdapter) Name() string { return a.inner.Name() }

func (a *credentialAdapter) FetchAll(ctx context.Context, config json.RawMessage) (ingestrecord.FetchResult, error) {
	cfg, err := usajobs.MergeCredentials(config, a.apiKey, a.userAgent)
	if err != nil {
		return ingestrecord.FetchResult{}, err
	}
	return a.inner.FetchAllWithConfig(ctx, cfg)
}
