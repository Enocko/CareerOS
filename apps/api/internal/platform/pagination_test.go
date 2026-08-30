package platform

import "testing"

func TestParsePagination(t *testing.T) {
	page, perPage := ParsePagination(0, 0)
	if page != 1 || perPage != 20 {
		t.Errorf("expected defaults 1, 20 got %d, %d", page, perPage)
	}

	page, perPage = ParsePagination(2, 150)
	if page != 2 || perPage != 100 {
		t.Errorf("expected 2, 100 got %d, %d", page, perPage)
	}
}

func TestNewPagination(t *testing.T) {
	p := NewPagination(1, 20, 45)
	if p.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", p.TotalPages)
	}
}
