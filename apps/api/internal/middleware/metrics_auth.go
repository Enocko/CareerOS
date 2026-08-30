package middleware

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/platform"
)

// MetricsAuth optionally protects the Prometheus metrics endpoint.
func MetricsAuth(token string) func(http.Handler) http.Handler {
	if token == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
				platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
