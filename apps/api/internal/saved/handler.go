package saved

import (
	"net/http"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for saved opportunities.
type Handler struct {
	service *Service
}

// NewHandler creates a new saved Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Save handles POST /api/v1/opportunities/:id/save.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	opportunityID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found"))
		return
	}

	saved, err := h.service.Save(r.Context(), userID, opportunityID)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, saved)
}

// Unsave handles DELETE /api/v1/opportunities/:id/save.
func (h *Handler) Unsave(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	opportunityID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Save not found"))
		return
	}

	if err := h.service.Unsave(r.Context(), userID, opportunityID); err != nil {
		platform.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/saved-opportunities.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	filter := ParseListFilter(r.URL.Query())
	result, err := h.service.List(r.Context(), userID, filter)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, result)
}
