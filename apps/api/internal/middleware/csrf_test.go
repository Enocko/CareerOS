package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/careeros/api/internal/auth"
)

func TestOriginMatchesAllowed(t *testing.T) {
	const allowed = "https://careeros-web.onrender.com"
	tests := []struct {
		origin string
		want   bool
	}{
		{"https://careeros-web.onrender.com", true},
		{"https://careeros-web.onrender.com/", true},
		{"https://careeros-web.onrender.com/opportunities?page=2", true},
		{"https://CareerOS-Web.onrender.com", true},
		{"https://careeros-web.onrender.com.evil.com", false},
		{"https://careeros-web.onrender.com.evil.com/x", false},
		{"https://careeros-web.onrender.com:8443", false},
		{"https://careeros-web.onrender.comx", false},
		{"http://careeros-web.onrender.com", false},
		{"https://evil.com/https://careeros-web.onrender.com", false},
		{"null", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := originMatchesAllowed(tt.origin, allowed); got != tt.want {
			t.Errorf("originMatchesAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}

	if originMatchesAllowed("https://careeros-web.onrender.com", "*") {
		t.Error("wildcard allowed origin must never match")
	}
	if originMatchesAllowed("https://careeros-web.onrender.com", "") {
		t.Error("empty allowed origin must never match")
	}
}

func TestCSRFOriginCheck(t *testing.T) {
	const allowed = "http://localhost:5173"
	handler := CSRFOriginCheck(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name    string
		method  string
		cookie  bool
		origin  string
		referer string
		want    int
	}{
		{"cookie POST from allowed origin", http.MethodPost, true, allowed, "", http.StatusNoContent},
		{"cookie POST from lookalike origin", http.MethodPost, true, "http://localhost:5173.evil.com", "", http.StatusForbidden},
		{"cookie POST with no origin", http.MethodPost, true, "", "", http.StatusForbidden},
		{"cookie POST with allowed referer", http.MethodPost, true, "", allowed + "/saved", http.StatusNoContent},
		{"cookie DELETE from other origin", http.MethodDelete, true, "https://evil.com", "", http.StatusForbidden},
		{"cookie GET from other origin", http.MethodGet, true, "https://evil.com", "", http.StatusNoContent},
		{"POST without session cookie", http.MethodPost, false, "https://evil.com", "", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/applications", nil)
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
