package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/careeros/api/internal/platform"
)

// Recovery catches panics and returns a 500 error instead of crashing the server.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", GetRequestID(r.Context()),
					"error", rec,
					"stack", string(debug.Stack()),
				)
				platform.WriteError(w, platform.InternalError())
			}
		}()

		next.ServeHTTP(w, r)
	})
}
