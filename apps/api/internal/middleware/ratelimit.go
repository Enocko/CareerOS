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
	mu        sync.Mutex
	limits    map[string]*rateBucket
	max       int
	window    time.Duration
	nextSweep time.Time
	now       func() time.Time
}

// NewRateLimiter creates a limiter allowing max requests per window per key.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rateBucket),
		max:    max,
		window: window,
		now:    time.Now,
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	rl.sweep(now)

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

// sweep drops expired buckets at most once per window so the map cannot grow
// without bound as new client IPs arrive. Caller must hold rl.mu.
func (rl *RateLimiter) sweep(now time.Time) {
	if now.Before(rl.nextSweep) {
		return
	}
	for key, b := range rl.limits {
		if now.After(b.resetAt) {
			delete(rl.limits, key)
		}
	}
	rl.nextSweep = now.Add(rl.window)
}

// LimitByIP rate-limits requests using the client IP as key. It relies on
// ClientIP having already resolved r.RemoteAddr; raw forwarding headers are
// never read here because clients can forge them.
func (rl *RateLimiter) LimitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(hostOnly(r.RemoteAddr)) {
			platform.WriteError(w, platform.NewAppError(http.StatusTooManyRequests, platform.ErrorCodeValidation, "Too many requests. Try again later."))
			return
		}
		next.ServeHTTP(w, r)
	})
}
