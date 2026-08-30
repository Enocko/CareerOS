package researchverification

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler serves admin research verification endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a research verification handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListQueue handles GET /api/v1/admin/research/queue
func (h *Handler) ListQueue(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	items, err := h.service.ListQueue(r.Context(), limit)
	if err != nil {
		platform.WriteError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

// GetOpportunity handles GET /api/v1/admin/research/opportunities/{id}
func (h *Handler) GetOpportunity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found"))
		return
	}
	item, err := h.service.GetOpportunity(r.Context(), id)
	if err != nil {
		platform.WriteError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, item)
}

// ListVerifications handles GET /api/v1/admin/research/opportunities/{id}/verifications
func (h *Handler) ListVerifications(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	records, err := h.service.ListVerifications(r.Context(), id, limit)
	if err != nil {
		platform.WriteError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]any{"data": records})
}

// Verify handles POST /api/v1/admin/research/opportunities/{id}/verify
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Research opportunity not found"))
		return
	}
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid request body"))
		return
	}
	rec, err := h.service.Verify(r.Context(), id, userID, req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, rec)
}

// GetMetrics handles GET /api/v1/admin/research/metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetMetrics(r.Context())
	if err != nil {
		platform.WriteError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, metrics)
}
