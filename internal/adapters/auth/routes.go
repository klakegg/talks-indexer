package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/javaBin/talks-indexer/internal/adapters/session"
	"github.com/javaBin/talks-indexer/internal/config"
)

// MiddlewareFunc is a function that wraps a handler with middleware
type MiddlewareFunc func(http.Handler) http.Handler

// Adapter holds the auth adapter dependencies
type Adapter struct {
	handler    *Handler
	middleware MiddlewareFunc
}

// passthroughMiddleware returns the handler unchanged (no authentication)
func passthroughMiddleware(next http.Handler) http.Handler {
	return next
}

// New creates a new auth adapter.
// In development mode, returns an adapter with passthrough middleware.
// In production mode, OIDC must be configured or an error is returned.
func New(ctx context.Context) (*Adapter, error) {
	cfg := config.GetConfig(ctx)

	// In development mode, use passthrough middleware (no auth required)
	if cfg.Mode.IsDevelopment() {
		slog.Info("auth disabled (development mode)")
		return &Adapter{
			middleware: passthroughMiddleware,
		}, nil
	}

	// In production, OIDC must be configured
	if !cfg.OIDC.IsConfigured() {
		return nil, fmt.Errorf("production mode but OIDC not configured")
	}

	// Set up OIDC authentication
	oidcConfig := OIDCConfig{
		IssuerURL:    cfg.OIDC.IssuerURL,
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		RedirectURL:  cfg.OIDC.RedirectURL,
	}

	authenticator, err := NewAuthenticator(ctx, oidcConfig)
	if err != nil {
		return nil, err
	}
	slog.Info("OIDC authenticator initialized")

	sessionStore := session.NewInMemoryStore()
	secureCookies := true

	authMiddleware := NewMiddleware(sessionStore, authenticator, secureCookies)
	authHandler := NewHandler(sessionStore, authenticator, secureCookies)

	return &Adapter{
		handler:    authHandler,
		middleware: authMiddleware.RequireAuth,
	}, nil
}

// RegisterRoutes registers auth routes (/auth/callback, /auth/logout).
// Only registers routes if OIDC authentication is enabled.
func (a *Adapter) RegisterRoutes(mux *http.ServeMux) {
	// Skip if no handler (development mode)
	if a.handler == nil {
		return
	}

	mux.HandleFunc("GET /auth/callback", a.handler.HandleCallback)
	mux.HandleFunc("POST /auth/logout", a.handler.HandleLogout)

	slog.Info("auth routes registered")
}

// Middleware returns the authentication middleware.
// In development mode, this is a passthrough (no-op) middleware.
// In production mode, this requires authentication.
func (a *Adapter) Middleware() MiddlewareFunc {
	return a.middleware
}
