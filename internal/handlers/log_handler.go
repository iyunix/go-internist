package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// FrontendLogPayload defines the structure for logs coming from the browser.
type FrontendLogPayload struct {
	Level   string `json:"level"`   // e.g., "info", "error", "warn"
	Message string `json:"message"` // The main log message
	Context any    `json:"context,omitempty"` // Optional extra data (e.g., stack trace)
}

// LogFrontendEvent handles incoming log requests from the frontend.
func LogFrontendEvent(w http.ResponseWriter, r *http.Request) {
	var payload FrontendLogPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// If decoding fails, we can't do much.
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use slog for structured, leveled logging on the server.
	// This makes logs much easier to query later.
	slog.Info("CLIENT_LOG",
		slog.String("level", payload.Level),
		slog.String("message", payload.Message),
		slog.Any("context", payload.Context),
	)

	// Respond with 204 No Content as we don't need to send anything back.
	w.WriteHeader(http.StatusNoContent)
}