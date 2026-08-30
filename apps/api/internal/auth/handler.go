package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/careeros/api/internal/platform"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service      *Service
	cookieConfig CookieConfig
	tokenExpiry  time.Duration
}

// NewHandler creates a new auth Handler.
func NewHandler(service *Service, cookieConfig CookieConfig, tokenExpiry time.Duration) *Handler {
	return &Handler{service: service, cookieConfig: cookieConfig, tokenExpiry: tokenExpiry}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	SetSessionCookie(w, resp.Token, h.tokenExpiry, h.cookieConfig)
	platform.WriteJSON(w, http.StatusCreated, resp)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	SetSessionCookie(w, resp.Token, h.tokenExpiry, h.cookieConfig)
	platform.WriteJSON(w, http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ClearSessionCookie(w, h.cookieConfig)
	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/v1/auth/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	resp, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusOK, resp)
}

// UserIDKey is the context key for the authenticated user's ID.
type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return id, ok
}

// UserEmailFromContext extracts the authenticated user email from context.
func UserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// WithUserEmail adds a user email to the context.
func WithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, UserEmailKey, email)
}
