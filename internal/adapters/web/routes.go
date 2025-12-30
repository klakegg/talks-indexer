package web

import (
	"net/http"

	"github.com/javaBin/talks-indexer/internal/adapters/web/handlers"
	"github.com/javaBin/talks-indexer/internal/ports"
)

// MiddlewareFunc is a function that wraps a handler with middleware
type MiddlewareFunc func(http.Handler) http.Handler

// Adapter holds the web adapter dependencies
type Adapter struct {
	handler *handlers.Handler
}

// New creates a new web adapter
func New(indexer ports.Indexer, provider ports.ConferenceProvider) *Adapter {
	return &Adapter{
		handler: handlers.NewHandler(indexer, provider),
	}
}

// RegisterRoutes registers all web routes with the provided mux.
// All routes are wrapped with the provided middleware (auth or passthrough).
func (a *Adapter) RegisterRoutes(mux *http.ServeMux, middleware MiddlewareFunc) {
	mux.Handle("GET /admin", middleware(http.HandlerFunc(a.handler.HandleDashboard)))
	mux.Handle("POST /admin/reindex/all", middleware(http.HandlerFunc(a.handler.HandleReindexAll)))
	mux.Handle("POST /admin/reindex/conference", middleware(http.HandlerFunc(a.handler.HandleReindexConference)))
	mux.Handle("POST /admin/reindex/talk", middleware(http.HandlerFunc(a.handler.HandleReindexTalk)))
}
