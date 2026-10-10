package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveClientIP(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		xff    []string
		hops   int
		want   string
	}{
		{"no proxy ignores forged header", "203.0.113.7:5000", []string{"1.2.3.4"}, 0, "203.0.113.7"},
		{"one hop uses entry appended by proxy", "10.0.0.1:443", []string{"9.9.9.9, 198.51.100.2"}, 1, "198.51.100.2"},
		{"forged left entries ignored", "10.0.0.1:443", []string{"6.6.6.6, 7.7.7.7, 198.51.100.2"}, 1, "198.51.100.2"},
		{"three hops", "10.0.0.3:443", []string{"6.6.6.6, 198.51.100.2, 10.0.0.1, 10.0.0.2"}, 3, "198.51.100.2"},
		{"multiple header lines", "10.0.0.1:443", []string{"6.6.6.6", "198.51.100.2"}, 1, "198.51.100.2"},
		{"short chain falls back to leftmost", "10.0.0.1:443", []string{"198.51.100.2"}, 3, "198.51.100.2"},
		{"garbage entry falls back to peer", "10.0.0.1:443", []string{"not-an-ip"}, 1, "10.0.0.1"},
		{"ipv6 peer", "[2001:db8::1]:443", nil, 0, "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveClientIP(tt.remote, tt.xff, tt.hops); got != tt.want {
				t.Fatalf("resolveClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLimitByIPIgnoresForgedForwardedFor(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	handler := ClientIP(0)(rl.LimitByIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	for i, want := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "203.0.113.7:1234"
		req.Header.Set("X-Forwarded-For", time.Now().Add(time.Duration(i)).String())
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("request %d: status = %d, want %d", i, rec.Code, want)
		}
	}
}

func TestRateLimiterSweepsExpiredBuckets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := NewRateLimiter(5, time.Minute)
	rl.now = func() time.Time { return now }

	for _, ip := range []string{"a", "b", "c"} {
		rl.allow(ip)
	}
	if len(rl.limits) != 3 {
		t.Fatalf("buckets = %d, want 3", len(rl.limits))
	}

	now = now.Add(2 * time.Minute)
	rl.allow("d")
	if len(rl.limits) != 1 {
		t.Fatalf("buckets after sweep = %d, want 1", len(rl.limits))
	}
}
