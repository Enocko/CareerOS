package recommendation_test

import (
	"sort"
	"testing"
	"time"

	"github.com/careeros/api/internal/recommendation"
)

// Evaluation fixture: relative ranking expectations for synthetic profiles.
func TestEvaluationFixtureRelativeRanking(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(14 * 24 * time.Hour)
	checked := now.Add(-24 * time.Hour)

	cases := []struct {
		name    string
		student recommendation.StudentContext
		items   []namedCandidate
		wantTop string
	}{
		{
			name: "CS student interested in SWE",
			student: recommendation.StudentContext{
				HasProfile: true, ProfileComplete: true,
				Major: "Computer Science",
				InferredFamilies: []string{"software_engineering"},
				PreferredExpLevels: []string{"internship"},
				Skills: []string{"python"},
				WorkArrangement: "remote",
			},
			items: []namedCandidate{
				{name: "swe", c: baseCandidate("Software Engineer Intern", "software_engineering", "internship", "remote", deadline, checked)},
				{name: "unrelated", c: baseCandidate("Technology Analyst Intern", "other_technical", "internship", "on_site", deadline, checked)},
			},
			wantTop: "swe",
		},
		{
			name: "Data science student",
			student: recommendation.StudentContext{
				HasProfile: true, ProfileComplete: true,
				CareerInterests: []string{"data science"},
				InferredFamilies: []string{"data_science"},
				Skills: []string{"python", "sql"},
			},
			items: []namedCandidate{
				{name: "ds", c: baseCandidate("Data Science Intern", "data_science", "internship", "hybrid", deadline, checked)},
				{name: "swe", c: baseCandidate("Software Engineer Intern", "software_engineering", "internship", "hybrid", deadline, checked)},
			},
			wantTop: "ds",
		},
		{
			name: "Cybersecurity student",
			student: recommendation.StudentContext{
				HasProfile: true, ProfileComplete: true,
				DesiredRoles: []string{"security engineer"},
				InferredFamilies: []string{"cybersecurity"},
			},
			items: []namedCandidate{
				{name: "sec", c: baseCandidate("Security Engineering Intern", "cybersecurity", "internship", "on_site", deadline, checked)},
				{name: "pm", c: baseCandidate("Product Management Intern", "product_management_technical", "internship", "on_site", deadline, checked)},
			},
			wantTop: "sec",
		},
		{
			name:    "Incomplete profile still ranks student roles",
			student: recommendation.StudentContext{},
			items: []namedCandidate{
				{name: "intern", c: baseCandidate("Software Engineer Intern", "software_engineering", "internship", "remote", deadline, checked)},
				{name: "senior", c: baseCandidate("Staff Engineer", "software_engineering", "unknown", "remote", deadline, checked)},
			},
			wantTop: "intern",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			type scored struct {
				name  string
				score int
			}
			var ranked []scored
			for _, item := range tc.items {
				s, _ := recommendation.Score(tc.student, item.c, now)
				ranked = append(ranked, scored{name: item.name, score: s})
			}
			sort.Slice(ranked, func(i, j int) bool {
				return ranked[i].score > ranked[j].score
			})
			if ranked[0].name != tc.wantTop {
				t.Fatalf("expected top %q, got %q (%+v)", tc.wantTop, ranked[0].name, ranked)
			}
		})
	}
}

type namedCandidate struct {
	name string
	c    recommendation.Candidate
}

func baseCandidate(title, family, exp, work string, deadline, checked time.Time) recommendation.Candidate {
	return recommendation.Candidate{
		Title: title, OrganizationName: "Fixture Co",
		WorkArrangement: work, Status: "open", VerificationStatus: "verified",
		CareerFamily: &family, ExperienceLevel: &exp,
		RelevanceTier: strPtr("high_confidence_technical"),
		Deadline: &deadline, LastCheckedAt: &checked,
	}
}

func strPtr(s string) *string { return &s }
