package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/careeros/api/internal/db"
	"github.com/careeros/api/internal/ingestion/ingestrecord"
	"github.com/google/uuid"
)

type stubAdapter struct {
	items    []ingestrecord.RawOpportunity
	err      error
	override *ingestrecord.FetchResult
}

func (s *stubAdapter) Name() string { return "stub" }

func (s *stubAdapter) FetchAll(ctx context.Context, config json.RawMessage) (ingestrecord.FetchResult, error) {
	if s.err != nil {
		return ingestrecord.FetchResult{}, s.err
	}
	if s.override != nil {
		return *s.override, nil
	}
	return ingestrecord.FetchResult{
		RawFetched:  len(s.items),
		Retained:    s.items,
		FilteredOut: 0,
	}.MarkExhaustiveSuccess(), nil
}

func exhaustiveEmptyFetch() ingestrecord.FetchResult {
	return ingestrecord.FetchResult{}.MarkExhaustiveSuccess()
}

func testPool(t *testing.T) *Repository {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable"
	}
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return NewRepository(pool)
}

func setupIsolatedTestSource(t *testing.T, repo *Repository) uuid.UUID {
	t.Helper()
	sourceID := uuid.New()
	ctx := context.Background()
	if err := repo.InsertTestSource(ctx, sourceID); err != nil {
		t.Fatalf("insert test source: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteTestSource(context.Background(), sourceID)
	})
	return sourceID
}

func cleanupTestOpportunity(t *testing.T, repo *Repository, sourceID uuid.UUID, externalID string) {
	t.Helper()
	t.Cleanup(func() {
		_ = repo.DeleteOpportunityByExternalID(context.Background(), sourceID, externalID)
	})
}

func TestFailedSyncDoesNotMarkStaleOrClosed(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "FAIL-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Existing Verified Role"); err != nil {
		t.Fatalf("insert test opportunity: %v", err)
	}

	statusBefore, missedBefore, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification before: %v", err)
	}
	if statusBefore != VerificationVerified {
		t.Fatalf("expected verified before failure, got %s", statusBefore)
	}

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{err: errors.New("upstream source timed out")}
		},
	})

	result, runErr := service.RunSource(ctx, sourceID)
	if runErr == nil {
		t.Fatal("expected ingestion error")
	}
	if result.Status != RunStatusFailed {
		t.Fatalf("expected failed run, got %s", result.Status)
	}

	statusAfter, missedAfter, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification after: %v", err)
	}
	if statusAfter != statusBefore {
		t.Errorf("verification_status changed from %s to %s", statusBefore, statusAfter)
	}
	if missedAfter != missedBefore {
		t.Errorf("missed_sync_count changed from %d to %d", missedBefore, missedAfter)
	}

	latest, err := repo.GetLatestRun(ctx, sourceID)
	if err != nil {
		t.Fatalf("get latest run: %v", err)
	}
	if latest == nil || latest.Status != RunStatusFailed {
		t.Fatalf("expected failed ingestion run, got %+v", latest)
	}
	if latest.RecordsStale != 0 || latest.RecordsClosed != 0 {
		t.Errorf("failed run should not record stale/closed counts, got stale=%d closed=%d", latest.RecordsStale, latest.RecordsClosed)
	}
}

func TestSuccessfulSyncMarksMissingAsStaleAfterTwoRuns(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "STALE-SYNC-TEST-" + uuid.NewString()
	decoyID := "STALE-SYNC-DECOY-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)
	cleanupTestOpportunity(t, repo, sourceID, decoyID)

	deadline := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	makeItem := func(id string) ingestrecord.RawOpportunity {
		return ingestrecord.RawOpportunity{
			ExternalID:      id,
			Title:           "Intern Role",
			Organization:    "NASA",
			Description:     "Student internship.",
			Category:        "internship",
			Location:        "Houston, TX",
			WorkArrangement: "on_site",
			Deadline:        &deadline,
			ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/" + id,
			SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/" + id,
			Tags:            []string{"federal"},
			Skills:          []string{},
		}
	}

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{makeItem(externalID)}}
		},
	})

	first, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if first.Status != RunStatusSuccess {
		t.Fatalf("initial sync expected success, got %s", first.Status)
	}

	service = NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{makeItem(decoyID)}}
		},
	})

	for i := 0; i < StaleAfterMissedSyncs; i++ {
		result, runErr := service.RunSource(ctx, sourceID)
		if runErr != nil {
			t.Fatalf("run %d failed: %v", i+1, runErr)
		}
		if result.Status != RunStatusSuccess {
			t.Fatalf("run %d expected success, got %s", i+1, result.Status)
		}
	}

	status, _, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationStale {
		t.Errorf("expected stale after %d missed syncs, got %s", StaleAfterMissedSyncs, status)
	}
}

func TestSuccessfulSyncUpsertsSeenOpportunities(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	deadline := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	externalID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{{
				ExternalID:      externalID,
				Title:           "Pathways Intern",
				Organization:    "NASA",
				Description:     "Student internship at NASA.",
				Category:        "internship",
				Location:        "Houston, TX",
				WorkArrangement: "on_site",
				Deadline:        &deadline,
				ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/upsert-test",
				SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/upsert-test",
				Tags:            []string{"federal"},
				Skills:          []string{},
			}}}
		},
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
	if result.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}
	if result.RecordsCreated < 1 && result.RecordsUpdated < 1 {
		t.Fatalf("expected created or updated record, got %+v", result)
	}

	status, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationVerified {
		t.Errorf("expected verified, got %s", status)
	}
	if missed != 0 {
		t.Errorf("expected missed_sync_count 0, got %d", missed)
	}
}

func TestResearchIngestionUpsertIdempotent(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedNSFSource(t, repo)

	externalID := "REU-UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	meta, _ := json.Marshal(map[string]any{
		"research_area":                      "Quantum Computing",
		"duration_weeks":                     10,
		"application_status":                 "unknown",
		"application_status_method":          "nsf_award_only",
		"availability_verification_method": "unknown",
	})
	service := NewService(repo, map[string]AdapterFactory{
		"nsf_reu": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{{
				ExternalID:         externalID,
				Title:              "REU Site: Quantum Computing",
				Organization:       "Test University",
				Description:         "NSF REU Site supporting undergraduate researchers for 10 weeks.",
				Category:           "research",
				Location:           "Boston, MA",
				ApplicationURL:     "https://example.edu/reu/apply",
				SourceURL:          "https://www.nsf.gov/awardsearch/showAward?AWD_ID=123",
				Tags:               []string{"nsf", "reu"},
				Skills:             []string{},
				OpportunityType:    "research",
				TypeMetadata:       meta,
				VerificationMethod: "official_source",
				EducationLevel:     "undergraduate",
			}}}
		},
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if result.RecordsCreated < 1 {
		t.Fatalf("expected created record, got %+v", result)
	}

	second, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.RecordsCreated != 0 {
		t.Errorf("expected no duplicates, created=%d", second.RecordsCreated)
	}
	if second.RecordsUpdated < 1 {
		t.Errorf("expected update on idempotent run, got %+v", second)
	}
}

func setupIsolatedNSFSource(t *testing.T, repo *Repository) uuid.UUID {
	t.Helper()
	sourceID := uuid.New()
	ctx := context.Background()
	config, err := json.Marshal(map[string]string{"base_url": "http://example.test"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := repo.InsertTestSourceWithConfig(ctx, sourceID, "nsf_reu", config); err != nil {
		t.Fatalf("insert test source: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteTestSource(context.Background(), sourceID)
	})
	return sourceID
}

func TestIsTestExternalID(t *testing.T) {
	if !IsTestExternalID("API-TEST-abc") {
		t.Error("expected API-TEST prefix to be test data")
	}
	if IsTestExternalID("MIL-26-13042786") {
		t.Error("expected real USAJobs ID not to be test data")
	}
}

type boardStubAdapter struct {
	failTokens   map[string]error
	successItems map[string][]ingestrecord.RawOpportunity
}

func (b *boardStubAdapter) Name() string { return "greenhouse" }

func (b *boardStubAdapter) FetchAll(ctx context.Context, config json.RawMessage) (ingestrecord.FetchResult, error) {
	var cfg struct {
		BoardToken string `json:"board_token"`
	}
	_ = json.Unmarshal(config, &cfg)
	if err, ok := b.failTokens[cfg.BoardToken]; ok {
		return ingestrecord.FetchResult{}, err
	}
	if items, ok := b.successItems[cfg.BoardToken]; ok {
		return ingestrecord.FetchResult{
			RawFetched:  len(items),
			Retained:    items,
			FilteredOut: 0,
		}.MarkExhaustiveSuccess(), nil
	}
	return exhaustiveEmptyFetch(), nil
}

func setupIsolatedGreenhouseBoard(t *testing.T, repo *Repository, boardToken string) uuid.UUID {
	t.Helper()
	sourceID := uuid.New()
	ctx := context.Background()
	config, err := json.Marshal(map[string]string{
		"board_token":   boardToken,
		"employer_name": "Test Employer",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := repo.InsertTestSourceWithConfig(ctx, sourceID, "greenhouse", config); err != nil {
		t.Fatalf("insert test source: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteTestSource(context.Background(), sourceID)
	})
	return sourceID
}

func TestPerBoardFailureDoesNotAffectOtherBoards(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()

	failSourceID := setupIsolatedGreenhouseBoard(t, repo, "fail-board")
	okSourceID := setupIsolatedGreenhouseBoard(t, repo, "ok-board")

	failExternalID := "GH-FAIL-BOARD-" + uuid.NewString()
	okExternalID := "GH-OK-BOARD-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, failSourceID, failExternalID)
	cleanupTestOpportunity(t, repo, okSourceID, okExternalID)

	if err := repo.InsertTestOpportunity(ctx, failSourceID, failExternalID, "Existing Verified Intern"); err != nil {
		t.Fatalf("insert fail-board opportunity: %v", err)
	}

	stub := &boardStubAdapter{
		failTokens: map[string]error{
			"fail-board": errors.New("board endpoint unavailable"),
		},
		successItems: map[string][]ingestrecord.RawOpportunity{
			"ok-board": {{
				ExternalID:      okExternalID,
				Title:           "Software Engineer Intern",
				Organization:    "Test Employer",
				Description:     "Internship role.",
				Category:        "internship",
				Location:        "Remote",
				WorkArrangement: "remote",
				ApplicationURL:  "https://boards.greenhouse.io/test/jobs/1",
				SourceURL:       "https://boards.greenhouse.io/test/jobs/1",
				Tags:            []string{"technology"},
				Skills:          []string{},
			}},
		},
	}

	service := NewService(repo, map[string]AdapterFactory{
		"greenhouse": func() Adapter { return stub },
	})

	failResult, failErr := service.RunSource(ctx, failSourceID)
	if failErr == nil {
		t.Fatal("expected fail-board ingestion error")
	}
	if failResult.Status != RunStatusFailed {
		t.Fatalf("expected failed run for fail-board, got %s", failResult.Status)
	}

	statusBefore, _, err := repo.GetOpportunityVerification(ctx, failSourceID, failExternalID)
	if err != nil {
		t.Fatalf("get fail-board verification: %v", err)
	}
	if statusBefore != VerificationVerified {
		t.Fatalf("expected fail-board listing to remain verified, got %s", statusBefore)
	}

	okResult, okErr := service.RunSource(ctx, okSourceID)
	if okErr != nil {
		t.Fatalf("ok-board ingestion failed: %v", okErr)
	}
	if okResult.Status != RunStatusSuccess {
		t.Fatalf("expected success for ok-board, got %s", okResult.Status)
	}
	if okResult.RecordsCreated < 1 {
		t.Fatalf("expected created record on ok-board, got %+v", okResult)
	}

	okStatus, _, err := repo.GetOpportunityVerification(ctx, okSourceID, okExternalID)
	if err != nil {
		t.Fatalf("get ok-board verification: %v", err)
	}
	if okStatus != VerificationVerified {
		t.Fatalf("expected ok-board listing verified, got %s", okStatus)
	}

	secondResult, secondErr := service.RunSource(ctx, okSourceID)
	if secondErr != nil {
		t.Fatalf("second ok-board run failed: %v", secondErr)
	}
	if secondResult.RecordsCreated != 0 {
		t.Errorf("expected no duplicates on second run, created=%d", secondResult.RecordsCreated)
	}
	if secondResult.RecordsUpdated < 1 {
		t.Errorf("expected update on second idempotent run, got %+v", secondResult)
	}
}

func TestIngestionMetricsPersisted(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{{
				ExternalID:      externalID,
				Title:           "Software Engineer Intern",
				Organization:    "Test Co",
				Description:     "Internship role.",
				Category:        "internship",
				Location:        "Remote",
				WorkArrangement: "remote",
				ApplicationURL:  "https://example.com/jobs/1",
				SourceURL:       "https://example.com/jobs/1",
				Tags:            []string{"technology"},
				Skills:          []string{},
			}}}
		},
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
	if result.RecordsRawFetched != 1 || result.RecordsRetained != 1 {
		t.Fatalf("expected raw/retained 1/1, got %d/%d", result.RecordsRawFetched, result.RecordsRetained)
	}

	latest, err := repo.GetLatestRun(ctx, sourceID)
	if err != nil {
		t.Fatalf("get latest run: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest run")
	}
	if latest.RecordsRawFetched != 1 || latest.RecordsRetained != 1 {
		t.Errorf("persisted metrics = raw %d retained %d", latest.RecordsRawFetched, latest.RecordsRetained)
	}
}

func TestSuspiciousEmptySyncDoesNotMarkStaleOrClosed(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "FAIL-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Existing Verified Role"); err != nil {
		t.Fatalf("insert test opportunity: %v", err)
	}

	emptyFetch := exhaustiveEmptyFetch()
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{override: &emptyFetch}
		},
	})

	result, runErr := service.RunSource(ctx, sourceID)
	if runErr == nil {
		t.Fatal("expected empty sync anomaly error")
	}
	if result.Status != RunStatusFailed {
		t.Fatalf("expected failed run, got %s", result.Status)
	}

	status, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationVerified {
		t.Errorf("expected listing to remain verified, got %s", status)
	}
	if missed != 0 {
		t.Errorf("expected missed_sync_count unchanged, got %d", missed)
	}

	latest, err := repo.GetLatestRun(ctx, sourceID)
	if err != nil {
		t.Fatalf("get latest run: %v", err)
	}
	if latest == nil || latest.ErrorCode == nil || *latest.ErrorCode != "empty_sync_anomaly" {
		t.Fatalf("expected empty_sync_anomaly, got %+v", latest)
	}
}

func TestRepeatedExhaustiveEmptySyncConfirmsGenuineEmptiness(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "STALE-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Will Become Stale"); err != nil {
		t.Fatalf("insert test opportunity: %v", err)
	}

	emptyFetch := exhaustiveEmptyFetch()
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{override: &emptyFetch}
		},
	})

	first, err := service.RunSource(ctx, sourceID)
	if err == nil {
		t.Fatal("expected first empty sync anomaly")
	}
	if first.Status != RunStatusFailed {
		t.Fatalf("expected failed first run, got %s", first.Status)
	}

	second, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("second empty sync: %v", err)
	}
	if second.Status != RunStatusSuccess {
		t.Fatalf("expected success on confirmed empty, got %s", second.Status)
	}

	status, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationVerified {
		t.Fatalf("expected verified after first confirmed miss, got %s", status)
	}
	if missed != 1 {
		t.Fatalf("expected missed_sync_count 1, got %d", missed)
	}
}

func TestFailedSyncBetweenEmptyAttemptsDoesNotConfirmEmptiness(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "FAIL-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)
	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Protected Role"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	emptyFetch := exhaustiveEmptyFetch()
	emptyService := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{override: &emptyFetch} },
	})
	if _, err := emptyService.RunSource(ctx, sourceID); err == nil {
		t.Fatal("expected first anomaly")
	}

	failService := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{err: errors.New("upstream unavailable")} },
	})
	if _, err := failService.RunSource(ctx, sourceID); err == nil {
		t.Fatal("expected fetch failure")
	}

	third, err := emptyService.RunSource(ctx, sourceID)
	if err == nil {
		t.Fatal("expected renewed anomaly after intervening failure")
	}
	if third.Status != RunStatusFailed {
		t.Fatalf("expected failed third run, got %s", third.Status)
	}

	_, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if missed != 0 {
		t.Fatalf("expected missed_sync_count unchanged, got %d", missed)
	}
}

func TestIncompleteEmptySyncDoesNotConfirmEmptiness(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "FAIL-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)
	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Protected Role"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	emptyFetch := exhaustiveEmptyFetch()
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{override: &emptyFetch} },
	})
	if _, err := service.RunSource(ctx, sourceID); err == nil {
		t.Fatal("expected first anomaly")
	}

	incomplete := ingestrecord.FetchResult{RawFetched: 0, Exhaustive: false}
	service = NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{override: &incomplete} },
	})
	result, err := service.RunSource(ctx, sourceID)
	if err == nil {
		t.Fatal("expected incomplete_sync failure")
	}
	if result.Status != RunStatusFailed {
		t.Fatalf("expected failed run, got %s", result.Status)
	}
	latest, err := repo.GetLatestRun(ctx, sourceID)
	if err != nil || latest.ErrorCode == nil || *latest.ErrorCode != "incomplete_sync" {
		t.Fatalf("expected incomplete_sync, got %+v", latest)
	}

	_, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if missed != 0 {
		t.Fatalf("expected missed_sync_count unchanged, got %d", missed)
	}
}

func TestRawPostingsWithoutRetainedDoesNotMassClose(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "FAIL-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)
	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Still Active Role"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	filteredOnly := ingestrecord.FetchResult{
		RawFetched: 5,
		Exhaustive: true,
	}
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{override: &filteredOnly} },
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("expected success without post-sync, got %v", err)
	}
	if result.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}

	status, missed, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationVerified || missed != 0 {
		t.Fatalf("expected verified/unchanged, got %s missed=%d", status, missed)
	}
}

func TestConfirmedEmptyBoardEventuallyStalesMissingListings(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "STALE-SYNC-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)
	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Will Become Stale"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	emptyFetch := exhaustiveEmptyFetch()
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter { return &stubAdapter{override: &emptyFetch} },
	})

	if _, err := service.RunSource(ctx, sourceID); err == nil {
		t.Fatal("expected first anomaly")
	}
	if _, err := service.RunSource(ctx, sourceID); err != nil {
		t.Fatalf("confirmed empty sync: %v", err)
	}

	for i := 0; i < StaleAfterMissedSyncs-1; i++ {
		result, err := service.RunSource(ctx, sourceID)
		if err != nil {
			t.Fatalf("ongoing empty sync %d: %v", i+1, err)
		}
		if result.Status != RunStatusSuccess {
			t.Fatalf("expected success on ongoing empty sync, got %s", result.Status)
		}
	}

	status, _, err := repo.GetOpportunityVerification(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get verification: %v", err)
	}
	if status != VerificationStale {
		t.Fatalf("expected stale after confirmed empty board, got %s", status)
	}
}

func TestAuthoritativeSyncReopensClosedOpportunity(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	deadline := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	externalID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	item := ingestrecord.RawOpportunity{
		ExternalID:      externalID,
		Title:           "Reopen Test Intern",
		Organization:    "NASA",
		Description:     "Student internship.",
		Category:        "internship",
		Location:        "Houston, TX",
		WorkArrangement: "on_site",
		Deadline:        &deadline,
		ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/reopen-test",
		SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/reopen-test",
		Tags:            []string{"federal"},
		Skills:          []string{},
	}

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{item}}
		},
	})

	first, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if first.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", first.Status)
	}

	oppID, _, _, err := repo.GetOpportunityLifecycle(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get lifecycle: %v", err)
	}

	_, err = repo.pool.Exec(ctx, `
		UPDATE opportunities
		SET status = 'closed', verification_status = $2, updated_at = now()
		WHERE id = $1
	`, oppID, VerificationClosed)
	if err != nil {
		t.Fatalf("mark closed: %v", err)
	}

	second, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("reopen sync: %v", err)
	}
	if second.Status != RunStatusSuccess {
		t.Fatalf("expected success on reopen sync, got %s", second.Status)
	}

	reopenedID, status, verification, err := repo.GetOpportunityLifecycle(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get lifecycle after reopen: %v", err)
	}
	if reopenedID != oppID {
		t.Fatalf("expected stable opportunity id %s, got %s", oppID, reopenedID)
	}
	if status != "open" || verification != VerificationVerified {
		t.Fatalf("expected open/verified after reopen, got %s/%s", status, verification)
	}
}

func TestFailedSyncDoesNotReopenClosedOpportunity(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	externalID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	if err := repo.InsertTestOpportunity(ctx, sourceID, externalID, "Closed Role"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	oppID, _, _, err := repo.GetOpportunityLifecycle(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get lifecycle: %v", err)
	}
	_, err = repo.pool.Exec(ctx, `
		UPDATE opportunities
		SET status = 'closed', verification_status = $2, updated_at = now()
		WHERE id = $1
	`, oppID, VerificationClosed)
	if err != nil {
		t.Fatalf("mark closed: %v", err)
	}

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{err: errors.New("upstream unavailable")}
		},
	})

	_, runErr := service.RunSource(ctx, sourceID)
	if runErr == nil {
		t.Fatal("expected fetch failure")
	}

	_, status, verification, err := repo.GetOpportunityLifecycle(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get lifecycle: %v", err)
	}
	if status != "closed" || verification != VerificationClosed {
		t.Fatalf("expected closed listing to remain closed, got %s/%s", status, verification)
	}
}

func TestDifferentExternalIDDoesNotReopenClosedOpportunity(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	closedID := "UPSERT-TEST-" + uuid.NewString()
	otherID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, closedID)
	cleanupTestOpportunity(t, repo, sourceID, otherID)

	if err := repo.InsertTestOpportunity(ctx, sourceID, closedID, "Similar Title Role"); err != nil {
		t.Fatalf("insert closed: %v", err)
	}
	oppID, _, _, err := repo.GetOpportunityLifecycle(ctx, sourceID, closedID)
	if err != nil {
		t.Fatalf("get lifecycle: %v", err)
	}
	_, err = repo.pool.Exec(ctx, `
		UPDATE opportunities
		SET status = 'closed', verification_status = $2, updated_at = now()
		WHERE id = $1
	`, oppID, VerificationClosed)
	if err != nil {
		t.Fatalf("mark closed: %v", err)
	}

	deadline := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{{
				ExternalID:      otherID,
				Title:           "Similar Title Role",
				Organization:    "NASA",
				Description:     "Different posting.",
				Category:        "internship",
				Location:        "Houston, TX",
				WorkArrangement: "on_site",
				Deadline:        &deadline,
				ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/other",
				SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/other",
				Tags:            []string{"federal"},
				Skills:          []string{},
			}}}
		},
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("sync other posting: %v", err)
	}
	if result.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}

	_, status, verification, err := repo.GetOpportunityLifecycle(ctx, sourceID, closedID)
	if err != nil {
		t.Fatalf("get closed lifecycle: %v", err)
	}
	if status != "closed" || verification != VerificationClosed {
		t.Fatalf("expected original closed listing unchanged, got %s/%s", status, verification)
	}
}

func TestSeenListingWithExpiredDeadlineStaysOpen(t *testing.T) {
	repo := testPool(t)
	ctx := context.Background()
	sourceID := setupIsolatedTestSource(t, repo)

	pastDeadline := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	externalID := "UPSERT-TEST-" + uuid.NewString()
	cleanupTestOpportunity(t, repo, sourceID, externalID)

	item := ingestrecord.RawOpportunity{
		ExternalID:      externalID,
		Title:           "Still Listed Intern",
		Organization:    "NASA",
		Description:     "Provider still lists role.",
		Category:        "internship",
		Location:        "Houston, TX",
		WorkArrangement: "on_site",
		Deadline:        &pastDeadline,
		ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/expired-deadline",
		SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/expired-deadline",
		Tags:            []string{"federal"},
		Skills:          []string{},
	}

	service := NewService(repo, map[string]AdapterFactory{
		"usajobs": func() Adapter {
			return &stubAdapter{items: []ingestrecord.RawOpportunity{item}}
		},
	})

	result, err := service.RunSource(ctx, sourceID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}

	_, status, verification, err := repo.GetOpportunityLifecycle(ctx, sourceID, externalID)
	if err != nil {
		t.Fatalf("get lifecycle: %v", err)
	}
	if status != "open" || verification != VerificationVerified {
		t.Fatalf("expected authoritative listing to remain open/verified, got %s/%s", status, verification)
	}
}
