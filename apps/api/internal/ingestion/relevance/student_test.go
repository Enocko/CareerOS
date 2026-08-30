package relevance

import "testing"

func TestClassifyStudentRelevance(t *testing.T) {
	tests := []struct {
		title  string
		want   bool
		reason string
	}{
		{"Software Engineer, Intern", true, ReasonInternshipTitleMatch},
		{"Software Engineer, Intern (Summer)", true, ReasonInternshipTitleMatch},
		{"Operations Associate, New Grad (Mexico)", true, ReasonNewGradMatch},
		{"Internal Audit Lead", false, ReasonNonRelevantRole},
		{"University Recruiter", false, ReasonRecruitingRoleExclusion},
		{"Senior Software Engineer", false, ReasonNonRelevantRole},
		{"Data Science Intern", true, ReasonInternshipTitleMatch},
		{"Cybersecurity Co-op", true, ReasonEarlyCareerMatch},
		{"Product Manager Intern", true, ReasonInternshipTitleMatch},
		{"Software Engineer, New Grad", true, ReasonNewGradMatch},
		{"Account Executive", false, ReasonNonRelevantRole},
		{"", false, ReasonEmptyTitle},
	}

	for _, tc := range tests {
		got := ClassifyStudentRelevance(tc.title)
		if got.Relevant != tc.want {
			t.Errorf("ClassifyStudentRelevance(%q).Relevant = %v, want %v", tc.title, got.Relevant, tc.want)
		}
		if got.Reason != tc.reason {
			t.Errorf("ClassifyStudentRelevance(%q).Reason = %q, want %q", tc.title, got.Reason, tc.reason)
		}
	}
}

func TestIsStudentRelevant(t *testing.T) {
	tests := []struct {
		title string
		want  bool
	}{
		{"Software Engineer, Intern", true},
		{"Software Engineer, Intern (Summer)", true},
		{"Operations Associate, New Grad (Mexico)", true},
		{"Internal Audit Lead", false},
		{"University Recruiter", false},
		{"Senior Software Engineer", false},
		{"Data Science Intern", true},
		{"Cybersecurity Co-op", true},
		{"Product Manager Intern", true},
		{"Software Engineer, New Grad", true},
		{"Account Executive", false},
	}

	for _, tc := range tests {
		got := IsStudentRelevant(tc.title)
		if got != tc.want {
			t.Errorf("IsStudentRelevant(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}
