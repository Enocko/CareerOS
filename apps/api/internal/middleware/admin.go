package middleware

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// RequireAdmin restricts access to configured admins.
//
// adminUserIDs is the preferred allowlist: a user ID only exists once its owner
// has registered, so it cannot be claimed by someone else. adminEmails is kept
// for compatibility, but because email ownership is not verified, an email must
// only be listed after its owner has registered with it.
func RequireAdmin(adminUserIDs map[uuid.UUID]struct{}, adminEmails map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, ok := auth.UserIDFromContext(r.Context()); ok {
				if _, allowed := adminUserIDs[userID]; allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			if email, ok := auth.UserEmailFromContext(r.Context()); ok && email != "" {
				if _, allowed := adminEmails[strings.ToLower(strings.TrimSpace(email))]; allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			platform.WriteError(w, platform.NewAppError(http.StatusForbidden, platform.ErrorCodeForbidden, "Admin access required"))
		})
	}
}
