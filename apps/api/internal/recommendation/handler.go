package recommendation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/middleware"
	"github.com/careeros/api/internal/observability"
	"github.com/careeros/api/internal/platform"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler serves recommendation HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a recommendation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type recommendResponse struct {
	Data       []Result       `json:"data"`
	Pagination platform.Pagination `json:"pagination"`
	Meta       RecommendMeta  `json:"meta"`
}

// List handles GET /api/v1/opportunities/recommended.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	start := time.Now()
	filter := parseListFilter(r.URL.Query())
	result, meta, err := h.service.Recommend(r.Context(), userID, filter)
	if err != nil {
		platform.WriteError(w, platform.InternalError())
		return
	}

	slog.Info("recommendation_impression",
		"request_id", middleware.GetRequestID(r.Context()),
		"student_id", userID.String(),
		"count", len(result.Data),
		"eligible_total", meta.EligibleCount,
		"profile_complete", meta.ProfileComplete,
		"duration_ms", time.Since(start).Milliseconds(),
		"outcome", "success",
	)
	observability.RecordRecommendationDuration(time.Since(start).Seconds())

	platform.WriteJSON(w, http.StatusOK, recommendResponse{
		Data:       result.Data,
		Pagination: result.Pagination,
		Meta:       *meta,
	})
}

// EventRequest is the payload for recommendation interaction events.
type EventRequest struct {
	Event         string `json:"event"`
	OpportunityID string `json:"opportunity_id,omitempty"`
}

// RecordEvent handles POST /api/v1/opportunities/recommended/events.
func (h *Handler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		platform.WriteError(w, platform.NewAppError(http.StatusUnauthorized, platform.ErrorCodeUnauthorized, "Authentication required"))
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid JSON body"))
		return
	}

	switch req.Event {
	case "recommendation_impression", "recommendation_opened",
		"opportunity_saved_from_recommendation", "official_application_link_clicked":
	default:
		platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Unsupported event type"))
		return
	}

	attrs := []any{
		"event", req.Event,
		"student_id", userID.String(),
	}
	if req.OpportunityID != "" {
		if _, err := uuid.Parse(req.OpportunityID); err != nil {
			platform.WriteError(w, platform.NewAppError(http.StatusBadRequest, platform.ErrorCodeValidation, "Invalid opportunity_id"))
			return
		}
		attrs = append(attrs, "opportunity_id", req.OpportunityID)
	}
	slog.Info("recommendation_event", attrs...)

	w.WriteHeader(http.StatusNoContent)
}

// OpenRecommended handles GET /api/v1/opportunities/recommended/{id} logging wrapper is optional;
// detail still uses standard opportunity endpoint.

func parseListFilter(values map[string][]string) ListFilter {
	get := func(key string) string {
		if v, ok := values[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	page, _ := strconv.Atoi(get("page"))
	perPage, _ := strconv.Atoi(get("per_page"))
	return ListFilter{Page: page, PerPage: perPage}
}

// RegisterRoutes mounts recommendation routes under /opportunities.
func RegisterRoutes(r chi.Router, h *Handler, authenticate func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(authenticate)
		r.Get("/recommended", h.List)
		r.Post("/recommended/events", h.RecordEvent)
	})
}
