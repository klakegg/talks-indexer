package api

import (
	"log/slog"
	"net/http"
)

// RegisterRoutes registers all API routes with the provided mux.
// Health check is always available. API routes are only registered in development mode.
func (a *Adapter) RegisterRoutes(mux *http.ServeMux) {
	// Health check is always available
	mux.HandleFunc("GET /health", a.HandleHealth)

	// API routes only available in development mode
	if a.cfg.Mode.IsDevelopment() {
		mux.HandleFunc("POST /api/reindex", a.HandleReindexAll)
		mux.HandleFunc("POST /api/reindex/conference/{slug}", a.HandleReindexConference)
		mux.HandleFunc("POST /api/reindex/talk/{talkId}", a.HandleReindexTalk)
		slog.Info("API routes enabled (development mode)")
	} else {
		slog.Info("API routes disabled (production mode)")
	}
}
