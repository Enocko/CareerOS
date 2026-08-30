package opportunities

import (
	"net/http"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for opportunities.
type Handler struct {
	service *Service
}

// NewHandler creates a new opportunities Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/opportunities.
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

// GetByID handles GET /api/v1/opportunities/:id.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	opportunityID, err := uuid.Parse(idStr)
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Opportunity not found"))
		return
	}

	opp, err := h.service.GetByID(r.Context(), userID, opportunityID)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, opp)
}
