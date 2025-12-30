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
	"github.com/javaBin/talks-indexer/internal/adapters/web"
	"github.com/javaBin/talks-indexer/internal/app"
	"github.com/javaBin/talks-indexer/internal/config"
)

func main() {
	// Load configuration first to determine logging mode
	cfg := config.MustLoad()

	// Inject config into context for use by adapters and services
	ctx := config.WithConfig(context.Background(), cfg)

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
	moresleepClient, err := moresleep.New(ctx)
	if err != nil {
		logger.Error("failed to create moresleep client", "error", err)
		os.Exit(1)
	}
	logger.Info("moresleep client initialized")

	// Initialize elasticsearch client
	esClient, err := elasticsearch.New(ctx)
	if err != nil {
		logger.Error("failed to create elasticsearch client", "error", err)
		os.Exit(1)
	}
	logger.Info("elasticsearch client initialized")

	// Create indexer service
	indexerService := app.NewIndexerService(
		ctx,
		moresleepClient,
		esClient,
		elasticsearch.TalkPrivateIndexMapping,
		elasticsearch.TalkPublicIndexMapping,
	)
	logger.Info("indexer service initialized")

	// Create HTTP server
	mux := http.NewServeMux()

	// Register API routes (mode-aware)
	apiAdapter := api.New(ctx, indexerService)
	apiAdapter.RegisterRoutes(mux)

	// Initialize auth adapter and register routes
	authAdapter, err := auth.New(ctx)
	if err != nil {
		logger.Error("failed to initialize auth", "error", err)
		os.Exit(1)
	}
	authAdapter.RegisterRoutes(mux)

	// Register web admin routes (protected if auth middleware is available)
	webAdapter := web.New(indexerService, moresleepClient)
	webAdapter.RegisterRoutes(mux, web.MiddlewareFunc(authAdapter.Middleware()))

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
