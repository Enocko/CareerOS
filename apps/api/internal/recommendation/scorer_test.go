package recommendation_test

import (
	"testing"
	"time"

	"github.com/careeros/api/internal/recommendation"
)

func TestSWEStudentRanksSWEAboveUnrelated(t *testing.T) {
	student := recommendation.StudentContext{
		HasProfile:       true,
		ProfileComplete:  true,
		InferredFamilies: []string{"software_engineering"},
		PreferredExpLevels: []string{"internship", "new_grad"},
		Skills:           []string{"python", "java"},
		WorkArrangement:  "remote",
	}
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

	swe := recommendation.Candidate{
		Title: "Software Engineer Intern", OrganizationName: "Acme",
		WorkArrangement: "remote", Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("software_engineering"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"), Skills: []string{"Python"},
		LastCheckedAt: ptrTime(now.Add(-24 * time.Hour)),
	}
	other := recommendation.Candidate{
		Title: "Technology Analyst Intern", OrganizationName: "Bank",
		WorkArrangement: "on_site", Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("other_technical"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"),
		LastCheckedAt: ptrTime(now.Add(-24 * time.Hour)),
	}

	sweScore, _ := recommendation.Score(student, swe, now)
	otherScore, _ := recommendation.Score(student, other, now)
	if sweScore <= otherScore {
		t.Fatalf("expected SWE (%d) > other technical (%d)", sweScore, otherScore)
	}
}

func TestDataScienceStudentRanksDSAboveSWE(t *testing.T) {
	student := recommendation.StudentContext{
		HasProfile:         true,
		ProfileComplete:    true,
		InferredFamilies:   []string{"data_science"},
		PreferredExpLevels: []string{"internship"},
		Skills:             []string{"python", "sql"},
	}
	now := time.Now().UTC()

	ds := recommendation.Candidate{
		Title: "Data Science Intern", WorkArrangement: "hybrid",
		Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("data_science"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"), Skills: []string{"Python", "SQL"},
	}
	swe := recommendation.Candidate{
		Title: "Software Engineer Intern", WorkArrangement: "hybrid",
		Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("software_engineering"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"),
	}

	dsScore, _ := recommendation.Score(student, ds, now)
	sweScore, _ := recommendation.Score(student, swe, now)
	if dsScore <= sweScore {
		t.Fatalf("expected data science (%d) > swe (%d)", dsScore, sweScore)
	}
}

func TestCybersecurityStudent(t *testing.T) {
	student := recommendation.StudentContext{
		HasProfile:       true,
		ProfileComplete:  true,
		InferredFamilies: []string{"cybersecurity"},
	}
	now := time.Now().UTC()
	sec := recommendation.Candidate{
		Title: "Security Engineering Intern", Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("cybersecurity"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"),
	}
	other := recommendation.Candidate{
		Title: "Marketing Intern", Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("non_technical"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_non_technical"),
	}
	secScore, _ := recommendation.Score(student, sec, now)
	otherScore, _ := recommendation.Score(student, other, now)
	if secScore <= otherScore {
		t.Fatalf("expected security (%d) > marketing (%d)", secScore, otherScore)
	}
}

func TestIncompleteProfileColdStart(t *testing.T) {
	student := recommendation.StudentContext{}
	now := time.Now().UTC()
	c := recommendation.Candidate{
		Title: "Software Engineer Intern", Status: "open", VerificationStatus: "verified",
		ExperienceLevel: ptr("internship"), RelevanceTier: ptr("high_confidence_technical"),
		LastCheckedAt: ptrTime(now.Add(-48 * time.Hour)),
	}
	score, factors := recommendation.Score(student, c, now)
	if score <= 0 {
		t.Fatalf("cold start should still produce positive score, got %d", score)
	}
	if len(factors) == 0 {
		t.Fatal("expected cold-start factors")
	}
}

func TestHardFilterExpiredDeadline(t *testing.T) {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := recommendation.Candidate{
		Status: "open", VerificationStatus: "verified",
		RelevanceTier: ptr("high_confidence_technical"),
		Deadline:      &past,
	}
	if recommendation.PassesHardFilters(c, time.Now().UTC()) {
		t.Fatal("expired deadline should be hard filtered")
	}
}

func TestHardFilterAppliedOpportunity(t *testing.T) {
	c := recommendation.Candidate{
		Status: "open", VerificationStatus: "verified",
		RelevanceTier: ptr("high_confidence_technical"), HasApplication: true,
	}
	if recommendation.PassesHardFilters(c, time.Now().UTC()) {
		t.Fatal("applied opportunities should be excluded")
	}
}

func TestScoreBoundaries(t *testing.T) {
	student := recommendation.StudentContext{
		HasProfile: true, ProfileComplete: true,
		InferredFamilies: []string{"software_engineering"},
		PreferredExpLevels: []string{"internship"},
		Skills: []string{"python"}, WorkArrangement: "remote",
		PreferredLocations: []string{"remote"},
	}
	now := time.Now().UTC()
	deadline := now.Add(5 * 24 * time.Hour)
	loc := "Remote"
	c := recommendation.Candidate{
		Title: "Software Engineer Intern", WorkArrangement: "remote",
		Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("software_engineering"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"), Skills: []string{"Python"},
		Location: &loc, Deadline: &deadline, LastCheckedAt: ptrTime(now.Add(-2 * time.Hour)),
	}
	score, _ := recommendation.Score(student, c, now)
	if score < 0 || score > 100 {
		t.Fatalf("score out of bounds: %d", score)
	}
}

func TestDeterministicOutput(t *testing.T) {
	student := recommendation.StudentContext{
		HasProfile: true, ProfileComplete: true,
		InferredFamilies: []string{"machine_learning_ai"},
		Skills: []string{"python"},
	}
	c := recommendation.Candidate{
		Title: "Machine Learning Intern", Status: "open", VerificationStatus: "verified",
		CareerFamily: ptr("machine_learning_ai"), ExperienceLevel: ptr("internship"),
		RelevanceTier: ptr("high_confidence_technical"),
	}
	now := time.Now().UTC()
	a, _ := recommendation.Score(student, c, now)
	b, _ := recommendation.Score(student, c, now)
	if a != b {
		t.Fatalf("expected deterministic score, got %d vs %d", a, b)
	}
}

func ptr(s string) *string { return &s }
func ptrTime(t time.Time) *time.Time { return &t }
