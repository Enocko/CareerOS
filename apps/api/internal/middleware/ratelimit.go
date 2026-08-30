package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/careeros/api/internal/platform"
)

type rateBucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter provides a simple in-process per-key rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	limits  map[string]*rateBucket
	max     int
	window  time.Duration
}

// NewRateLimiter creates a limiter allowing max requests per window per key.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rateBucket),
		max:    max,
		window: window,
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.limits[key]
	if !ok || now.After(b.resetAt) {
		rl.limits[key] = &rateBucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.max {
		return false
	}
	b.count++
	return true
}

// LimitByIP rate-limits requests using the client IP as key.
func (rl *RateLimiter) LimitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}
		if !rl.allow(ip) {
			platform.WriteError(w, platform.NewAppError(http.StatusTooManyRequests, platform.ErrorCodeValidation, "Too many requests. Try again later."))
			return
		}
		next.ServeHTTP(w, r)
	})
}
