package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/javaBin/talks-indexer/internal/adapters/api"
	"github.com/javaBin/talks-indexer/internal/adapters/auth"
	"github.com/javaBin/talks-indexer/internal/adapters/elasticsearch"
	"github.com/javaBin/talks-indexer/internal/adapters/moresleep"
	"github.com/javaBin/talks-indexer/internal/adapters/session"
	webAdapter "github.com/javaBin/talks-indexer/internal/adapters/web"
	"github.com/javaBin/talks-indexer/internal/adapters/web/handlers"
	"github.com/javaBin/talks-indexer/internal/app"
	"github.com/javaBin/talks-indexer/internal/config"
)

func main() {
	// Load configuration first to determine logging mode
	cfg := config.MustLoad()

	// Configure logging based on mode
	var logger *slog.Logger
	if cfg.Mode.IsDevelopment() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	slog.SetDefault(logger)

	logger.Info("configuration loaded",
		"mode", cfg.Mode,
		"httpAddr", cfg.Http.Addr(),
		"moresleepURL", cfg.Moresleep.URL,
		"elasticsearchURL", cfg.Elasticsearch.URL,
		"privateIndex", cfg.Index.Private,
		"publicIndex", cfg.Index.Public,
	)

	// Initialize moresleep client
	moresleepClient := moresleep.New(
		cfg.Moresleep.URL,
		cfg.Moresleep.User,
		cfg.Moresleep.Password,
	)
	logger.Info("moresleep client initialized")

	// Initialize elasticsearch client
	esClient, err := elasticsearch.New(
		cfg.Elasticsearch.URL,
		cfg.Elasticsearch.User,
		cfg.Elasticsearch.Password,
	)
	if err != nil {
		logger.Error("failed to create elasticsearch client", "error", err)
		os.Exit(1)
	}
	logger.Info("elasticsearch client initialized")

	// Create indexer service
	indexerService := app.NewIndexerService(
		moresleepClient,
		esClient,
		cfg.Index.Private,
		cfg.Index.Public,
		elasticsearch.TalkPrivateIndexMapping,
		elasticsearch.TalkPublicIndexMapping,
	)
	logger.Info("indexer service initialized")

	// Create HTTP server
	mux := http.NewServeMux()

	// Health check is always available
	apiHandler := api.NewHandler(indexerService)
	api.RegisterHealthRoutes(mux, apiHandler)

	// API routes only available in development mode
	if cfg.Mode.IsDevelopment() {
		api.RegisterAPIRoutes(mux, apiHandler)
		logger.Info("API routes enabled (development mode)")
	} else {
		logger.Info("API routes disabled (production mode)")
	}

	// Web admin dashboard
	webHandler := handlers.NewHandler(indexerService, moresleepClient)

	// Set up authentication in production mode
	if !cfg.Mode.IsDevelopment() && cfg.OIDC.IsConfigured() {
		oidcConfig := auth.OIDCConfig{
			IssuerURL:    cfg.OIDC.IssuerURL,
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  cfg.OIDC.RedirectURL,
		}

		authenticator, err := auth.NewAuthenticator(context.Background(), oidcConfig)
		if err != nil {
			logger.Error("failed to create OIDC authenticator", "error", err)
			os.Exit(1)
		}
		logger.Info("OIDC authenticator initialized")

		sessionStore := session.NewInMemoryStore()
		secureCookies := true

		authMiddleware := auth.NewMiddleware(sessionStore, authenticator, secureCookies)
		authHandler := auth.NewHandler(sessionStore, authenticator, secureCookies)

		mux.HandleFunc("GET /auth/callback", authHandler.HandleCallback)
		mux.HandleFunc("POST /auth/logout", authHandler.HandleLogout)

		webAdapter.RegisterProtectedRoutes(mux, webHandler, authMiddleware)
		logger.Info("admin routes protected with OIDC authentication")
	} else {
		webAdapter.RegisterRoutes(mux, webHandler)
		if !cfg.Mode.IsDevelopment() && !cfg.OIDC.IsConfigured() {
			logger.Warn("production mode but OIDC not configured - admin routes unprotected")
		}
	}

	server := &http.Server{
		Addr:         cfg.Http.Addr(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for reindex operations
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
