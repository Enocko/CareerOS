package nsf_reu

import "testing"

func TestClassifyURLsFromAbstract(t *testing.T) {
	abstract := `Apply via https://etap.nsf.gov. Program info at https://example.edu/reu/apply and https://www.nsf.gov/awardsearch/showAward?AWD_ID=1`
	links := classifyURLsFromAbstract(abstract)
	if links.ApplicationURL != "" {
		t.Errorf("expected no application URL, got %q", links.ApplicationURL)
	}
	if links.ProgramURL != "https://example.edu/reu/apply" {
		t.Errorf("program_url = %q", links.ProgramURL)
	}
}

func TestIsGenericETAPURL(t *testing.T) {
	if !isGenericETAPURL("https://etap.nsf.gov/") {
		t.Fatal("expected generic ETAP root")
	}
	if isGenericETAPURL("https://etap.nsf.gov/opportunity/123") {
		t.Fatal("specific ETAP path is not generic root")
	}
}

func TestIsNSFAwardURL(t *testing.T) {
	if !isNSFAwardURL("https://www.nsf.gov/awardsearch/showAward?AWD_ID=1") {
		t.Fatal("expected NSF award URL detection")
	}
}
