package catalogreport_test

import (
	"testing"

	"github.com/careeros/api/internal/catalogreport"
)

func TestClassifyURL(t *testing.T) {
	tests := []struct {
		url    string
		issue  string
	}{
		{"https://localhost/jobs/1", "localhost/test host"},
		{"https://etap.nsf.gov/", "generic ETAP homepage"},
		{"https://boards.greenhouse.io/acme", "generic Greenhouse board"},
		{"ftp://example.com", "missing http(s) scheme"},
		{"https://jobs.ashbyhq.com/acme/uuid", ""},
	}
	for _, tc := range tests {
		got := catalogreport.ClassifyURLForTest(tc.url)
		if got != tc.issue {
			t.Fatalf("classifyURL(%q) = %q, want %q", tc.url, got, tc.issue)
		}
	}
}
