package middleware

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
)

// RequireAdmin restricts access to configured admin email addresses.
func RequireAdmin(adminEmails map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email, ok := auth.UserEmailFromContext(r.Context())
			if !ok || email == "" {
				platform.WriteError(w, platform.NewAppError(http.StatusForbidden, platform.ErrorCodeForbidden, "Admin access required"))
				return
			}
			if _, allowed := adminEmails[strings.ToLower(strings.TrimSpace(email))]; !allowed {
				platform.WriteError(w, platform.NewAppError(http.StatusForbidden, platform.ErrorCodeForbidden, "Admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
