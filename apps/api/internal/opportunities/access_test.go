package opportunities

import (
	"testing"

	"github.com/careeros/api/internal/ingestion"
	"github.com/google/uuid"
)

func TestIsBrowseDiscoverable(t *testing.T) {
	tier := "high_confidence_technical"
	ambiguous := "ambiguous"

	tests := []struct {
		name       string
		opp        Opportunity
		externalID string
		want       bool
	}{
		{
			name: "open verified employment",
			opp: Opportunity{
				Status:             "open",
				VerificationStatus: ingestion.VerificationVerified,
				OpportunityType:    "employment",
				RelevanceTier:      &tier,
			},
			want: true,
		},
		{
			name: "closed employment",
			opp: Opportunity{
				Status:             "closed",
				VerificationStatus: ingestion.VerificationClosed,
				OpportunityType:    "employment",
				RelevanceTier:      &tier,
			},
			want: false,
		},
		{
			name: "stale employment",
			opp: Opportunity{
				Status:             "open",
				VerificationStatus: ingestion.VerificationStale,
				OpportunityType:    "employment",
				RelevanceTier:      &tier,
			},
			want: false,
		},
		{
			name: "ambiguous employment hidden by default",
			opp: Opportunity{
				Status:             "open",
				VerificationStatus: ingestion.VerificationVerified,
				OpportunityType:    "employment",
				RelevanceTier:      &ambiguous,
			},
			want: false,
		},
		{
			name: "test fixture excluded",
			opp: Opportunity{
				Status:             "open",
				VerificationStatus: ingestion.VerificationVerified,
				OpportunityType:    "employment",
				RelevanceTier:      &tier,
			},
			externalID: "API-TEST-" + uuid.NewString(),
			want:       false,
		},
		{
			name: "open verified research",
			opp: Opportunity{
				Status:             "open",
				VerificationStatus: ingestion.VerificationVerified,
				OpportunityType:    "research",
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBrowseDiscoverable(&tc.opp, tc.externalID); got != tc.want {
				t.Fatalf("isBrowseDiscoverable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanAccessDetail(t *testing.T) {
	tier := "high_confidence_technical"
	closed := Opportunity{
		Status:             "closed",
		VerificationStatus: ingestion.VerificationClosed,
		OpportunityType:    "employment",
		RelevanceTier:      &tier,
	}

	if canAccessDetail(&closed, "", false) {
		t.Fatal("expected closed listing without relationship to be inaccessible")
	}

	closed.IsSaved = true
	if !canAccessDetail(&closed, "", false) {
		t.Fatal("expected saved closed listing to be accessible")
	}

	closed.IsSaved = false
	closed.HasApplication = true
	if !canAccessDetail(&closed, "", false) {
		t.Fatal("expected tracked closed listing to be accessible")
	}

	closed.HasApplication = false
	if !canAccessDetail(&closed, "", true) {
		t.Fatal("expected previously viewed closed listing to be accessible")
	}
}
