package nsf_reu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleAwardJSON = `{
  "response": {
    "award": [{
      "id": "2548170",
      "title": "REU Site: Microbiology at the host-pathogen interface",
      "abstractText": "This REU Site award to The University of Iowa will support 10 students for 10 weeks during the summers of 2027-2029. Students will apply using NSF ETAP (https://etap.nsf.gov). More information at https://microbiology.medicine.uiowa.edu/undergraduate-education/research-opportunities/summer-undergraduate-research.",
      "awardeeName": "University of Iowa",
      "awardeeCity": "IOWA CITY",
      "awardeeStateCode": "IA",
      "startDate": "03/01/2027",
      "expDate": "02/28/2030",
      "program": "REU SITE-Res Exp for Ugrd Site",
      "orgLongName": "Directorate for Biological Sciences",
      "fundProgramName": "RSCH EXPER FOR UNDERGRAD SITES",
      "activeAwd": "true"
    }],
    "metadata": {"offset": 0, "rpp": 25, "totalCount": 1}
  }
}`

func TestFetchAllValidAward(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ActiveAwards") != "True" {
			t.Errorf("expected ActiveAwards=True, got %q", r.URL.Query().Get("ActiveAwards"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleAwardJSON))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	result, err := adapter.FetchAll(context.Background(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if result.RawFetched != 1 || len(result.Retained) != 1 {
		t.Fatalf("expected 1 retained, got raw=%d retained=%d", result.RawFetched, len(result.Retained))
	}

	raw := result.Retained[0]
	if raw.ExternalID != "2548170" {
		t.Errorf("external_id = %q", raw.ExternalID)
	}
	if raw.OpportunityType != "research" {
		t.Errorf("opportunity_type = %q", raw.OpportunityType)
	}
	if raw.VerificationMethod != "official_source" {
		t.Errorf("verification_method = %q", raw.VerificationMethod)
	}
	if raw.ApplicationURL != "" {
		t.Errorf("application_url should be empty for award-only discovery, got %q", raw.ApplicationURL)
	}
	var meta map[string]any
	if err := json.Unmarshal(raw.TypeMetadata, &meta); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if meta["application_status"] != "unknown" {
		t.Errorf("application_status = %v", meta["application_status"])
	}
	programURL, _ := meta["program_url"].(string)
	if !strings.Contains(programURL, "uiowa.edu") {
		t.Errorf("expected program_url in metadata, got %q", programURL)
	}
	if raw.Deadline != nil {
		t.Fatal("deadline must not be inferred from NSF award data")
	}
}

func TestFetchAllMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	_, err := adapter.FetchAll(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFetchAllAPIErrorNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"serviceNotification":[{"notificationType":"ERROR","notificationMessage":"bad param"}]}}`))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	_, err := adapter.FetchAll(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "bad param") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestFetchAllMissingRequiredFields(t *testing.T) {
	body := `{
	  "response": {
	    "award": [{
	      "id": "1",
	      "title": "REU Site: Test",
	      "abstractText": "",
	      "awardeeName": "Test U",
	      "expDate": "02/28/2030"
	    }],
	    "metadata": {"totalCount": 1}
	  }
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	result, err := adapter.FetchAll(context.Background(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(result.Retained) != 0 {
		t.Fatalf("expected filtered record, retained=%d reasons=%v", len(result.Retained), result.FilterReasons)
	}
	if result.FilterReasons["missing_description"] != 1 {
		t.Errorf("filter reasons = %v", result.FilterReasons)
	}
}

func TestFetchAllExpiredAwardFiltered(t *testing.T) {
	body := `{
	  "response": {
	    "award": [{
	      "id": "2",
	      "title": "REU Site: Old Program",
	      "abstractText": "Program description with https://example.edu/apply.",
	      "awardeeName": "Example University",
	      "expDate": "01/01/2020"
	    }],
	    "metadata": {"totalCount": 1}
	  }
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	result, err := adapter.FetchAll(context.Background(), cfg)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(result.Retained) != 0 {
		t.Fatal("expected expired award to be filtered")
	}
	if result.FilterReasons["expired_award"] != 1 {
		t.Errorf("filter reasons = %v", result.FilterReasons)
	}
}

func TestFetchAllTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(map[string]string{"base_url": srv.URL})
	adapter := NewAdapter(srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 50)
	defer cancel()
	_, err := adapter.FetchAll(ctx, cfg)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestNormalizeAwardMetadata(t *testing.T) {
	award := nsfAward{
		Title:        "REU Site: Quantum Computing",
		AbstractText: "Support 8 students for 10 weeks.",
		StartDate:    "06/01/2028",
		ExpDate:      "05/31/2031",
		OrgLongName:  "Directorate for Mathematical and Physical Sciences",
	}
	exp, _ := parseNSFDate(award.ExpDate)
	meta := buildTypeMetadata(award, exp, classifiedURLs{})
	if meta.ResearchArea == nil || *meta.ResearchArea != "Quantum Computing" {
		t.Fatalf("research_area = %v", meta.ResearchArea)
	}
	if meta.DurationWeeks == nil || *meta.DurationWeeks != 10 {
		t.Fatalf("duration_weeks = %v", meta.DurationWeeks)
	}
}
