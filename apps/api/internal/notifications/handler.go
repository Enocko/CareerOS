package notifications

import (
	"net/http"
	"strconv"
	"time"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler serves notification HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a notification handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /api/v1/notifications.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	page, perPage = platform.ParsePagination(page, perPage)
	offset := (page - 1) * perPage

	items, err := h.service.List(r.Context(), userID, perPage, offset)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}

	unread, err := h.service.UnreadCount(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}

	platform.WriteJSON(w, http.StatusOK, map[string]any{
		"data":    items,
		"unread":  unread,
		"pagination": platform.Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   len(items),
		},
	})
}

// UnreadCount handles GET /api/v1/notifications/unread-count.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}
	count, err := h.service.UnreadCount(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]int{"unread": count})
}

// MarkRead handles PATCH /api/v1/notifications/{id}/read.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Notification not found"))
		return
	}
	if err := h.service.MarkRead(r.Context(), userID, id, time.Now().UTC()); err != nil {
		if err == ErrNotFound {
			platform.WriteError(w, platform.NewAppError(http.StatusNotFound, platform.ErrorCodeNotFound, "Notification not found"))
			return
		}
		platform.WriteError(w, platform.InternalError())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead handles POST /api/v1/notifications/mark-all-read.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}
	count, err := h.service.MarkAllRead(r.Context(), userID, time.Now().UTC())
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]int64{"marked_read": count})
}
