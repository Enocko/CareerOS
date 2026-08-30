package platform

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

// WriteError writes a standard API error response.
func WriteError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.Status, map[string]any{
			"error": APIError{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
		return
	}

	WriteJSON(w, http.StatusInternalServerError, map[string]any{
		"error": APIError{
			Code:    ErrorCodeInternal,
			Message: "An unexpected error occurred",
		},
	})
}
