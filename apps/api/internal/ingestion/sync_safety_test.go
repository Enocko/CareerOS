package ingestion

import (
	"testing"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
)

func strPtr(s string) *string { return &s }

func TestDecidePostSync_NormalRetention(t *testing.T) {
	decision := DecidePostSync(5, 2, ingestrecord.FetchResult{RawFetched: 2}, nil)
	if !decision.ApplyPostSync || decision.FailRun {
		t.Fatalf("expected normal post-sync, got %+v", decision)
	}
}

func TestDecidePostSync_NewSourceEmpty(t *testing.T) {
	fetch := ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
	decision := DecidePostSync(0, 0, fetch, nil)
	if !decision.ApplyPostSync || decision.FailRun {
		t.Fatalf("expected noop post-sync for new empty source, got %+v", decision)
	}
}

func TestDecidePostSync_FirstUnexpectedEmpty(t *testing.T) {
	fetch := ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
	decision := DecidePostSync(3, 0, fetch, nil)
	if decision.ApplyPostSync || !decision.FailRun || decision.ErrorCode != "empty_sync_anomaly" {
		t.Fatalf("expected first empty anomaly, got %+v", decision)
	}
}

func TestDecidePostSync_SecondConsecutiveEmptyConfirms(t *testing.T) {
	fetch := ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
	prev := &Run{
		Status:            RunStatusFailed,
		ErrorCode:         strPtr("empty_sync_anomaly"),
		RecordsRawFetched: 0,
		RecordsRetained:   0,
	}
	decision := DecidePostSync(3, 0, fetch, prev)
	if !decision.ApplyPostSync || decision.FailRun {
		t.Fatalf("expected confirmed empty post-sync, got %+v", decision)
	}
}

func TestDecidePostSync_RawWithoutRetainedSkipsAbsence(t *testing.T) {
	fetch := ingestrecord.FetchResult{
		RawFetched: 4,
		Exhaustive: true,
	}
	decision := DecidePostSync(3, 0, fetch, nil)
	if decision.ApplyPostSync || decision.FailRun {
		t.Fatalf("expected skip without failure for raw>0 retained=0, got %+v", decision)
	}
}

func TestDecidePostSync_IncompleteDoesNotConfirm(t *testing.T) {
	fetch := ingestrecord.FetchResult{RawFetched: 0}
	prev := &Run{
		Status:            RunStatusFailed,
		ErrorCode:         strPtr("empty_sync_anomaly"),
		RecordsRawFetched: 0,
		RecordsRetained:   0,
	}
	decision := DecidePostSync(2, 0, fetch, prev)
	if decision.ApplyPostSync || decision.ErrorCode != "incomplete_sync" {
		t.Fatalf("expected incomplete_sync, got %+v", decision)
	}
}

func TestDecidePostSync_EstablishedEmptyBoardContinues(t *testing.T) {
	fetch := ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
	prev := &Run{
		Status:            RunStatusSuccess,
		RecordsRawFetched: 0,
		RecordsRetained:   0,
	}
	decision := DecidePostSync(2, 0, fetch, prev)
	if !decision.ApplyPostSync || decision.FailRun {
		t.Fatalf("expected ongoing empty board post-sync, got %+v", decision)
	}
}

func TestDecidePostSync_FailedSyncBetweenDoesNotConfirm(t *testing.T) {
	fetch := ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
	prev := &Run{
		Status:    RunStatusFailed,
		ErrorCode: strPtr("fetch_failed"),
	}
	decision := DecidePostSync(2, 0, fetch, prev)
	if decision.ApplyPostSync || decision.ErrorCode != "empty_sync_anomaly" {
		t.Fatalf("expected renewed anomaly after unrelated failure, got %+v", decision)
	}
}
