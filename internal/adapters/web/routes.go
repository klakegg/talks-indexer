package web

import (
	"net/http"

	"github.com/javaBin/talks-indexer/internal/adapters/auth"
	"github.com/javaBin/talks-indexer/internal/adapters/web/handlers"
)

// RegisterRoutes registers all web routes with the provided mux (no auth)
func RegisterRoutes(mux *http.ServeMux, h *handlers.Handler) {
	// Admin dashboard
	mux.HandleFunc("GET /admin", h.HandleDashboard)

	// htmx endpoints for reindex operations
	mux.HandleFunc("POST /admin/reindex/all", h.HandleReindexAll)
	mux.HandleFunc("POST /admin/reindex/conference", h.HandleReindexConference)
	mux.HandleFunc("POST /admin/reindex/talk", h.HandleReindexTalk)
}

// RegisterProtectedRoutes registers admin routes protected by auth middleware
func RegisterProtectedRoutes(mux *http.ServeMux, h *handlers.Handler, authMiddleware *auth.Middleware) {
	// Create handlers wrapped with auth middleware
	protectedDashboard := authMiddleware.RequireAuth(http.HandlerFunc(h.HandleDashboard))
	protectedReindexAll := authMiddleware.RequireAuth(http.HandlerFunc(h.HandleReindexAll))
	protectedReindexConf := authMiddleware.RequireAuth(http.HandlerFunc(h.HandleReindexConference))
	protectedReindexTalk := authMiddleware.RequireAuth(http.HandlerFunc(h.HandleReindexTalk))

	// Register protected routes
	mux.Handle("GET /admin", protectedDashboard)
	mux.Handle("POST /admin/reindex/all", protectedReindexAll)
	mux.Handle("POST /admin/reindex/conference", protectedReindexConf)
	mux.Handle("POST /admin/reindex/talk", protectedReindexTalk)
}
