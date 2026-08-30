package ingestion

import (
	"testing"
	"time"

	"github.com/careeros/api/internal/ingestion/ingestrecord"
)

func TestNormalizeRawRequiresURLs(t *testing.T) {
	_, ok := NormalizeRaw(ingestrecord.RawOpportunity{
		ExternalID:   "1",
		Title:        "Intern",
		Organization: "NASA",
		Description:  "desc",
	})
	if ok {
		t.Fatal("expected invalid without URLs")
	}

	normalized, ok := NormalizeRaw(ingestrecord.RawOpportunity{
		ExternalID:     "1",
		Title:          "Intern",
		Organization:   "NASA",
		Description:    "desc",
		ApplicationURL: "https://www.usajobs.gov/job/1",
		SourceURL:      "https://www.usajobs.gov/job/1",
	})
	if !ok {
		t.Fatal("expected valid opportunity")
	}
	if normalized.ApplicationURL != normalized.SourceURL {
		t.Fatalf("expected mirrored URLs, got app=%s source=%s", normalized.ApplicationURL, normalized.SourceURL)
	}
}

func TestNormalizeRawAcceptsUSAJobsFixtureShape(t *testing.T) {
	deadline := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	normalized, ok := NormalizeRaw(ingestrecord.RawOpportunity{
		ExternalID:      "TEST-POS-001",
		Title:           "Student Trainee (Information Technology)",
		Organization:    "Department of Veterans Affairs",
		Description:     "Assist with IT systems support.",
		Category:        "internship",
		Location:        "Washington, DC",
		WorkArrangement: "hybrid",
		Deadline:        &deadline,
		ApplicationURL:  "https://www.usajobs.gov/GetJob/ViewDetails/10000001",
		SourceURL:       "https://www.usajobs.gov/GetJob/ViewDetails/10000001",
		Tags:            []string{"federal"},
		Skills:          []string{},
	})
	if !ok {
		t.Fatal("expected fixture-shaped item to normalize")
	}
	if normalized.Category != "internship" || normalized.WorkArrangement != "hybrid" {
		t.Fatalf("unexpected normalized fields: %+v", normalized)
	}
}
