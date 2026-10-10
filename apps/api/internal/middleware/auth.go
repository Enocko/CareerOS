package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
)

// RevocationChecker reports whether a session token was revoked by logout.
type RevocationChecker interface {
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}

// Authenticate validates JWT tokens from HttpOnly cookies or Authorization header.
// Tokens without an ID (issued before revocation support) or revoked by logout
// are rejected.
func Authenticate(tokens *auth.TokenManager, revocations RevocationChecker) func(http.Handler) http.Handler {
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
			if err != nil || claims.ID == "" {
				platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Invalid or expired token"))
				return
			}

			revoked, err := revocations.IsTokenRevoked(r.Context(), claims.ID)
			if err != nil {
				slog.Error("check token revocation", "error", err)
				platform.WriteError(w, platform.InternalError())
				return
			}
			if revoked {
				platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Invalid or expired token"))
				return
			}

			ctx := auth.WithUserID(r.Context(), claims.UserID)
			ctx = auth.WithUserEmail(ctx, claims.Email)
			session := auth.Session{TokenID: claims.ID}
			if claims.ExpiresAt != nil {
				session.ExpiresAt = claims.ExpiresAt.Time
			}
			ctx = auth.WithSession(ctx, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
