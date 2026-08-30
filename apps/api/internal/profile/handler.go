package profile

import (
	"encoding/json"
	"net/http"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
)

// Handler handles HTTP requests for student profiles.
type Handler struct {
	service *Service
}

// NewHandler creates a new profile Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Get handles GET /api/v1/profile.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, profile)
}

// Update handles PUT /api/v1/profile.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	profile, err := h.service.UpsertProfile(r.Context(), userID, req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, profile)
}
