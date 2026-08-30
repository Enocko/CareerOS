package ingestion

import (
	"fmt"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
)

// EmptySyncGuardMinVerifiedOpen is the minimum number of verified open opportunities
// already stored for a source before zero-retained syncs are evaluated for emptiness risk.
const EmptySyncGuardMinVerifiedOpen = 1

// EmptySyncConfirmationsRequired is the number of consecutive exhaustive empty syncs
// (after an initial anomaly) required before post-sync absence processing runs.
// The first unexpected exhaustive empty always records empty_sync_anomaly; the second
// consecutive exhaustive empty with no intervening non-empty sync confirms the board.
const EmptySyncConfirmationsRequired = 2

// PostSyncDecision describes whether post-sync stale/close actions should run.
type PostSyncDecision struct {
	ApplyPostSync bool
	FailRun       bool
	ErrorCode     string
	Message       string
}

// DecidePostSync evaluates whether post-sync actions are safe for this fetch outcome.
func DecidePostSync(
	verifiedOpen int,
	retainedCount int,
	fetch ingestrecord.FetchResult,
	prevFinished *Run,
) PostSyncDecision {
	if retainedCount > 0 {
		return PostSyncDecision{ApplyPostSync: true}
	}

	if verifiedOpen < EmptySyncGuardMinVerifiedOpen {
		return PostSyncDecision{ApplyPostSync: true}
	}

	// Source has verified listings but this sync retained none.
	if fetch.RawFetched > 0 {
		return PostSyncDecision{
			ApplyPostSync: false,
			FailRun:       false,
			Message:       "retained zero relevant postings; no per-listing absence signal",
		}
	}

	// raw == 0, retained == 0, verifiedOpen >= 1
	if !fetch.Exhaustive {
		return PostSyncDecision{
			ApplyPostSync: false,
			FailRun:       true,
			ErrorCode:     "incomplete_sync",
			Message:       "sync did not establish an exhaustive source result",
		}
	}

	if authoritativeEmptyConfirmed(fetch, prevFinished) {
		return PostSyncDecision{ApplyPostSync: true}
	}

	return PostSyncDecision{
		ApplyPostSync: false,
		FailRun:       true,
		ErrorCode:     "empty_sync_anomaly",
		Message: fmt.Sprintf(
			"sync retained 0 opportunities but source has %d verified open listings",
			verifiedOpen,
		),
	}
}

func authoritativeEmptyConfirmed(fetch ingestrecord.FetchResult, prevFinished *Run) bool {
	if fetch.AuthoritativeEmpty && prevFinished != nil && wasPendingEmptyConfirmation(prevFinished) {
		return true
	}
	if prevFinished != nil && wasEstablishedEmptyBoardSync(prevFinished) && fetch.AuthoritativeEmpty {
		return true
	}
	return false
}

func wasPendingEmptyConfirmation(run *Run) bool {
	if run.Status != RunStatusFailed || run.ErrorCode == nil {
		return false
	}
	return *run.ErrorCode == "empty_sync_anomaly" &&
		run.RecordsRawFetched == 0 &&
		run.RecordsRetained == 0
}

func wasEstablishedEmptyBoardSync(run *Run) bool {
	return run.Status == RunStatusSuccess &&
		run.RecordsRawFetched == 0 &&
		run.RecordsRetained == 0
}
