package api

import (
	"net/http"
)

// RegisterHealthRoutes registers the health check endpoint (always available)
func RegisterHealthRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /health", h.HandleHealth)
}

// RegisterAPIRoutes registers API routes (development mode only)
func RegisterAPIRoutes(mux *http.ServeMux, h *Handler) {
	// Reindex endpoints
	mux.HandleFunc("POST /api/reindex", h.HandleReindexAll)
	mux.HandleFunc("POST /api/reindex/conference/{slug}", h.HandleReindexConference)
	mux.HandleFunc("POST /api/reindex/talk/{talkId}", h.HandleReindexTalk)
}

// RegisterRoutes registers all HTTP routes with the provided mux
// Deprecated: Use RegisterHealthRoutes and RegisterAPIRoutes separately
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	RegisterHealthRoutes(mux, h)
	RegisterAPIRoutes(mux, h)
}
