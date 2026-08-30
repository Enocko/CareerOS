package v2_test

import (
	"testing"

	v2 "github.com/careeros/api/internal/ingestion/relevance/v2"
)

func TestCorpusShouldBeTechnical(t *testing.T) {
	titles := []string{
		"Software Engineer Intern",
		"Software Engineering Intern",
		"Backend Engineering Intern",
		"Machine Learning Intern",
		"Data Science Intern",
		"Security Engineering Intern",
		"Technical Product Management Intern",
		"Software Engineer, New Grad",
		"Associate Software Engineer",
		"Software Engineer I",
		"Engineering Co-op",
		"Developer Intern",
		"Research Engineer Intern",
		"Applied Scientist Intern",
		"Graduate Software Engineer",
		"Entry Level Engineer",
		"Technology Analyst Intern",
	}

	for _, title := range titles {
		c := v2.Classify(title, "")
		if !c.InTechnicalFeed {
			t.Errorf("%q: expected InTechnicalFeed=true, got tier=%s family=%s reasons=%v",
				title, c.RelevanceTier, c.CareerFamily, c.Reasons)
		}
		if c.RelevanceTier != v2.TierHighConfidenceTechnical {
			t.Errorf("%q: expected high_confidence_technical, got %s", title, c.RelevanceTier)
		}
	}
}

func TestCorpusShouldBeNonTechnical(t *testing.T) {
	titles := []string{
		"Marketing Intern",
		"SDR Intern",
		"Sales Intern",
		"Customer Experience Associate (New Grad)",
		"Recruiting Intern",
		"Finance Intern",
		"Product Marketing Intern",
	}

	for _, title := range titles {
		c := v2.Classify(title, "")
		if c.RelevanceTier != v2.TierHighConfidenceNonTechnical {
			t.Errorf("%q: expected high_confidence_non_technical, got %s (family=%s)", title, c.RelevanceTier, c.CareerFamily)
		}
		if c.InTechnicalFeed {
			t.Errorf("%q: should not be in technical feed", title)
		}
	}
}

func TestAmbiguousCases(t *testing.T) {
	cases := []struct {
		title string
		tier  v2.RelevanceTier
	}{
		{"Operations Associate, New Grad (Mexico)", v2.TierAmbiguous},
		{"Business Analyst Intern", v2.TierAmbiguous},
		{"General Intern", v2.TierAmbiguous},
	}

	for _, tc := range cases {
		c := v2.Classify(tc.title, "")
		if c.RelevanceTier != tc.tier {
			t.Errorf("%q: expected tier %s, got %s (family=%s)", tc.title, tc.tier, c.RelevanceTier, c.CareerFamily)
		}
		if c.InTechnicalFeed {
			t.Errorf("%q: ambiguous case should not auto-enter technical feed", tc.title)
		}
	}
}

func TestExperienceLevels(t *testing.T) {
	cases := []struct {
		title string
		level v2.ExperienceLevel
	}{
		{"Software Engineer Intern", v2.ExperienceInternship},
		{"Engineering Co-op", v2.ExperienceCoOp},
		{"Software Engineer, New Grad", v2.ExperienceNewGrad},
		{"Early Career Engineer", v2.ExperienceEarlyCareer},
		{"Apprenticeship Program", v2.ExperienceApprenticeship},
		{"Research Fellowship", v2.ExperienceFellowship},
		{"Associate Software Engineer", v2.ExperienceEarlyCareer},
	}

	for _, tc := range cases {
		c := v2.Classify(tc.title, "")
		if c.ExperienceLevel != tc.level {
			t.Errorf("%q: expected experience %s, got %s", tc.title, tc.level, c.ExperienceLevel)
		}
	}
}

func TestPhDResearchInternship(t *testing.T) {
	c := v2.Classify("PhD Research Intern", "Doctoral candidates only. PhD required.")
	if c.EducationLevel != v2.EducationPhD {
		t.Fatalf("expected phd education level, got %s", c.EducationLevel)
	}
	if c.InTechnicalFeed {
		t.Error("phd research internship should not enter primary undergrad technical feed")
	}
	found := false
	for _, r := range c.Reasons {
		if r == "phd_research_internship" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected phd_research_internship reason, got %v", c.Reasons)
	}
}

func TestShouldPersistSourceRecord(t *testing.T) {
	persistCases := []string{
		"Marketing Intern",
		"Software Engineer Intern",
		"Associate Software Engineer",
		"Software Engineer I",
	}
	for _, title := range persistCases {
		c := v2.Classify(title, "")
		if !v2.ShouldPersistSourceRecord(title, c) {
			t.Errorf("%q should persist at ingest", title)
		}
	}

	skipCases := []string{
		"Senior Software Engineer",
		"University Recruiter",
		"Account Executive",
		"",
	}
	for _, title := range skipCases {
		c := v2.Classify(title, "")
		if v2.ShouldPersistSourceRecord(title, c) {
			t.Errorf("%q should not persist at ingest", title)
		}
	}
}

func TestProductManagementVsProductMarketing(t *testing.T) {
	pm := v2.Classify("Product Management Intern", "")
	if pm.CareerFamily != v2.CareerProductManagementTechnical {
		t.Errorf("Product Management Intern: expected technical product, got %s", pm.CareerFamily)
	}

	pmm := v2.Classify("Product Marketing Intern", "")
	if pmm.CareerFamily != v2.CareerNonTechnical {
		t.Errorf("Product Marketing Intern: expected non_technical, got %s", pmm.CareerFamily)
	}
}

func TestSalesforceEngineerNotExcluded(t *testing.T) {
	c := v2.Classify("Salesforce Software Engineer Intern", "")
	if c.CareerFamily == v2.CareerNonTechnical {
		t.Errorf("Salesforce engineer intern should not be classified as non_technical sales, got %s", c.CareerFamily)
	}
}

func TestStaffPrincipalEngineerINotEntryLevel(t *testing.T) {
	titles := []string{
		"Staff Software Engineer I - SRE",
		"Principal Software Engineer I - Metadata",
		"Senior Software Engineer I — Agentic Analytics",
	}
	for _, title := range titles {
		c := v2.Classify(title, "")
		if v2.ShouldPersistSourceRecord(title, c) {
			t.Errorf("%q should not persist as student/entry-level", title)
		}
	}
}

func TestSoftwareEngineerIIsEntryLevel(t *testing.T) {
	c := v2.Classify("Software Engineer I, Backend", "")
	if c.ExperienceLevel != v2.ExperienceEarlyCareer {
		t.Fatalf("expected early_career, got %s", c.ExperienceLevel)
	}
	if !v2.ShouldPersistSourceRecord("Software Engineer I, Backend", c) {
		t.Fatal("Software Engineer I should persist")
	}
}

func TestPrimaryReasonCode(t *testing.T) {
	c := v2.Classify("Data Science Intern", "")
	code := c.PrimaryReasonCode()
	if code != "internship + data_science" {
		t.Errorf("expected internship + data_science, got %q", code)
	}
}
