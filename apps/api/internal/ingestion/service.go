package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/careeros/api/internal/opportunitytype"
	"github.com/google/uuid"
)

// Service orchestrates opportunity ingestion from registered sources.
type Service struct {
	repo     *Repository
	adapters map[string]AdapterFactory
}

// NewService creates a new ingestion Service.
func NewService(repo *Repository, adapters map[string]AdapterFactory) *Service {
	return &Service{repo: repo, adapters: adapters}
}

// RunAll ingests from all enabled sources sequentially.
func (s *Service) RunAll(ctx context.Context) ([]RunResult, error) {
	sources, err := s.repo.ListEnabledSources(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]RunResult, 0, len(sources))
	for _, source := range sources {
		result, runErr := s.RunSource(ctx, source.ID)
		if runErr != nil {
			slog.Error("ingestion source failed", "source", source.Name, "error", runErr)
		}
		results = append(results, result)
	}
	return results, nil
}

// RunSource ingests opportunities from a single source.
func (s *Service) RunSource(ctx context.Context, sourceID uuid.UUID) (RunResult, error) {
	source, err := s.repo.GetSourceByID(ctx, sourceID)
	if err != nil {
		return RunResult{}, err
	}

	factory, ok := s.adapters[source.Adapter]
	if !ok {
		return RunResult{}, fmt.Errorf("no adapter registered for %q", source.Adapter)
	}

	runID, err := s.repo.CreateRun(ctx, source.ID)
	if err != nil {
		return RunResult{}, err
	}

	result := RunResult{
		RunID:      runID,
		SourceName: source.Name,
		Status:     RunStatusFailed,
	}

	adapter := factory()
	now := time.Now().UTC()

	fetchResult, fetchErr := adapter.FetchAll(ctx, source.Config)
	if fetchErr != nil {
		errMsg := fetchErr.Error()
		errCode := classifyError(fetchErr)
		if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, FinishCounts{}, &errMsg, &errCode); finishErr != nil {
			return result, finishErr
		}
		result.ErrorMessage = errMsg
		return result, fetchErr
	}

	seenExternalIDs := make([]string, 0, len(fetchResult.Retained))
	created, updated := 0, 0

	for _, item := range fetchResult.Retained {
		normalized, ok := NormalizeRaw(item)
		if !ok {
			continue
		}
		if !opportunitytype.IsNonEmploymentIngestion(normalized.OpportunityType) {
			ApplyClassification(&normalized)
		}

		isCreated, upsertErr := s.repo.UpsertOpportunity(ctx, source.ID, source.Name, normalized, now)
		if upsertErr != nil {
			errMsg := upsertErr.Error()
			errCode := "upsert_failed"
			if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, FinishCounts{
				RawFetched:  fetchResult.RawFetched,
				Retained:    fetchResult.RetainedCount(),
				FilteredOut: fetchResult.FilteredOut,
				Created:     created,
				Updated:     updated,
			}, &errMsg, &errCode); finishErr != nil {
				return result, finishErr
			}
			result.ErrorMessage = errMsg
			return result, upsertErr
		}

		seenExternalIDs = append(seenExternalIDs, normalized.ExternalID)
		if isCreated {
			created++
		} else {
			updated++
		}
	}

	verifiedOpen, countErr := s.repo.CountVerifiedBySource(ctx, source.ID)
	if countErr != nil {
		errMsg := countErr.Error()
		errCode := "post_sync_failed"
		if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, FinishCounts{
			RawFetched:  fetchResult.RawFetched,
			Retained:    fetchResult.RetainedCount(),
			FilteredOut: fetchResult.FilteredOut,
			Created:     created,
			Updated:     updated,
		}, &errMsg, &errCode); finishErr != nil {
			return result, finishErr
		}
		result.ErrorMessage = errMsg
		return result, countErr
	}

	prevFinished, prevErr := s.repo.GetPreviousFinishedRun(ctx, source.ID, runID)
	if prevErr != nil {
		errMsg := prevErr.Error()
		errCode := "post_sync_failed"
		if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, FinishCounts{
			RawFetched:  fetchResult.RawFetched,
			Retained:    fetchResult.RetainedCount(),
			FilteredOut: fetchResult.FilteredOut,
			Created:     created,
			Updated:     updated,
		}, &errMsg, &errCode); finishErr != nil {
			return result, finishErr
		}
		result.ErrorMessage = errMsg
		return result, prevErr
	}

	postSyncDecision := DecidePostSync(verifiedOpen, len(seenExternalIDs), fetchResult, prevFinished)
	if !postSyncDecision.ApplyPostSync {
		counts := FinishCounts{
			RawFetched:  fetchResult.RawFetched,
			Retained:    fetchResult.RetainedCount(),
			FilteredOut: fetchResult.FilteredOut,
			Created:     created,
			Updated:     updated,
		}
		if postSyncDecision.FailRun {
			errMsg := postSyncDecision.Message
			errCode := postSyncDecision.ErrorCode
			if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, counts, &errMsg, &errCode); finishErr != nil {
				return result, finishErr
			}
			result.Status = RunStatusFailed
			result.ErrorMessage = errMsg
			slog.Warn("ingestion post-sync blocked",
				"source", source.Name,
				"verified_open", verifiedOpen,
				"raw_fetched", fetchResult.RawFetched,
				"retained", fetchResult.RetainedCount(),
				"error_code", errCode,
			)
			return result, fmt.Errorf("%s", errMsg)
		}

		if finishErr := s.repo.FinishRun(ctx, runID, RunStatusSuccess, counts, nil, nil); finishErr != nil {
			return result, finishErr
		}
		result.Status = RunStatusSuccess
		result.RecordsRawFetched = counts.RawFetched
		result.RecordsRetained = counts.Retained
		result.RecordsFilteredOut = counts.FilteredOut
		result.RecordsFetched = counts.Retained
		result.RecordsCreated = created
		result.RecordsUpdated = updated
		slog.Info("ingestion completed without post-sync",
			"source", source.Name,
			"reason", postSyncDecision.Message,
			"raw_fetched", fetchResult.RawFetched,
			"retained", fetchResult.RetainedCount(),
		)
		return result, nil
	}

	staleCount, closedCount, postErr := s.repo.ApplyPostSyncActions(ctx, source.ID, seenExternalIDs, now)
	if postErr != nil {
		errMsg := postErr.Error()
		errCode := "post_sync_failed"
		if finishErr := s.repo.FinishRun(ctx, runID, RunStatusFailed, FinishCounts{
			RawFetched:  fetchResult.RawFetched,
			Retained:    fetchResult.RetainedCount(),
			FilteredOut: fetchResult.FilteredOut,
			Created:     created,
			Updated:     updated,
		}, &errMsg, &errCode); finishErr != nil {
			return result, finishErr
		}
		result.ErrorMessage = errMsg
		return result, postErr
	}

	counts := FinishCounts{
		RawFetched:  fetchResult.RawFetched,
		Retained:    fetchResult.RetainedCount(),
		FilteredOut: fetchResult.FilteredOut,
		Created:     created,
		Updated:     updated,
		Stale:       staleCount,
		Closed:      closedCount,
	}
	if finishErr := s.repo.FinishRun(ctx, runID, RunStatusSuccess, counts, nil, nil); finishErr != nil {
		return result, finishErr
	}

	result.Status = RunStatusSuccess
	result.RecordsRawFetched = counts.RawFetched
	result.RecordsRetained = counts.Retained
	result.RecordsFilteredOut = counts.FilteredOut
	result.RecordsFetched = counts.Retained
	result.RecordsCreated = created
	result.RecordsUpdated = updated
	result.RecordsStale = staleCount
	result.RecordsClosed = closedCount

	slog.Info("ingestion completed",
		"source", source.Name,
		"raw_fetched", result.RecordsRawFetched,
		"retained", result.RecordsRetained,
		"filtered_out", result.RecordsFilteredOut,
		"filter_reasons", fetchResult.FilterReasons,
		"created", result.RecordsCreated,
		"updated", result.RecordsUpdated,
		"stale", result.RecordsStale,
		"closed", result.RecordsClosed,
	)

	return result, nil
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "timeout", "deadline exceeded", "context canceled"):
		return "timeout"
	case containsAny(msg, "429", "rate limit"):
		return "rate_limited"
	case containsAny(msg, "401", "403", "unauthorized"):
		return "auth_failed"
	default:
		return "fetch_failed"
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(strings.ToLower(s), strings.ToLower(p)) {
			return true
		}
	}
	return false
}
