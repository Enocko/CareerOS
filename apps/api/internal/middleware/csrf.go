package middleware

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
)

// CSRFOriginCheck validates the Origin header on state-changing requests when
// cookie-based session auth is in use. Bearer-only API clients skip this check.
//
// SameSite=Lax cookies provide baseline CSRF protection for cross-site POSTs.
// Origin validation adds defense when frontend and API share a configured origin.
func CSRFOriginCheck(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isStateChanging(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}

			if auth.SessionTokenFromRequest(r) == "" {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if origin == "" || !originMatchesAllowed(origin, allowedOrigin) {
				platform.WriteError(w, platform.NewAppError(http.StatusForbidden, platform.ErrorCodeUnauthorized, "Invalid request origin"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isStateChanging(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func originMatchesAllowed(origin, allowed string) bool {
	if allowed == "" || allowed == "*" {
		return false
	}
	origin = strings.TrimRight(origin, "/")
	allowed = strings.TrimRight(allowed, "/")
	return strings.HasPrefix(origin, allowed)
}
