package platform

import (
	"net/http"

	"github.com/careeros/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthResponse is returned by GET /health (liveness).
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadinessResponse is returned by GET /ready.
type ReadinessResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// LivenessHandler reports process liveness only.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	}
}

// ReadinessHandler reports dependency readiness (PostgreSQL).
func ReadinessHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ReadinessResponse{
			Status:   "ready",
			Database: "connected",
		}
		status := http.StatusOK

		if pool == nil || db.Ping(r.Context(), pool) != nil {
			resp.Status = "not_ready"
			resp.Database = "disconnected"
			status = http.StatusServiceUnavailable
		}

		WriteJSON(w, status, resp)
	}
}

// HealthHandler is kept for backward compatibility; use LivenessHandler + ReadinessHandler.
func HealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return ReadinessHandler(pool)
}
