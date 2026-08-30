package applications

import (
	"encoding/json"
	"net/http"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for applications.
type Handler struct {
	service *Service
}

// NewHandler creates a new applications Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /api/v1/applications.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	app, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusCreated, app)
}

// List handles GET /api/v1/applications.
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

// GetByID handles GET /api/v1/applications/:id.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	applicationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found"))
		return
	}

	app, err := h.service.GetByID(r.Context(), userID, applicationID)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, app)
}

// Update handles PATCH /api/v1/applications/:id.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	applicationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found"))
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	app, err := h.service.Update(r.Context(), userID, applicationID, req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, app)
}

// Delete handles DELETE /api/v1/applications/:id.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	applicationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Application not found"))
		return
	}

	if err := h.service.Delete(r.Context(), userID, applicationID); err != nil {
		platform.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
