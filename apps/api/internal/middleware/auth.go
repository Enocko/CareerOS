package middleware

import (
	"net/http"
	"strings"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
)

// Authenticate validates JWT tokens from HttpOnly cookies or Authorization header.
func Authenticate(tokens *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.SessionTokenFromRequest(r)
			if token == "" {
				header := r.Header.Get("Authorization")
				if header != "" {
					parts := strings.SplitN(header, " ", 2)
					if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
						token = parts[1]
					}
				}
			}

			if token == "" {
				platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
				return
			}

			claims, err := tokens.Validate(token)
			if err != nil {
				platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Invalid or expired token"))
				return
			}

			ctx := auth.WithUserID(r.Context(), claims.UserID)
			ctx = auth.WithUserEmail(ctx, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
