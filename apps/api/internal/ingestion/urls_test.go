package ingestion

import "testing"

func TestIsPublicHTTPURL(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"https://jobs.example.com/apply", true},
		{"http://www.usajobs.gov/job/1", true},
		{"https://localhost/jobs", false},
		{"http://127.0.0.1:8080/x", false},
		{"ftp://example.com", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isPublicHTTPURL(tc.raw); got != tc.want {
			t.Fatalf("isPublicHTTPURL(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}
