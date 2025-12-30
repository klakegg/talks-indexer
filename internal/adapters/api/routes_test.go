package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/stretchr/testify/assert"
)

// testConfigDevelopment creates a test config in development mode
func testConfigDevelopment() *config.Config {
	return &config.Config{
		ApplicationConfig: config.ApplicationConfig{
			Mode: config.ModeDevelopment,
		},
	}
}

// testConfigProduction creates a test config in production mode
func testConfigProduction() *config.Config {
	return &config.Config{
		ApplicationConfig: config.ApplicationConfig{
			Mode: config.ModeProduction,
		},
	}
}

func TestRegisterRoutes_DevelopmentMode(t *testing.T) {
	ctx := config.WithConfig(context.Background(), testConfigDevelopment())
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)
	mux := http.NewServeMux()

	adapter.RegisterRoutes(mux)

	// Test that all routes are registered in development mode
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /health",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/reindex",
			method:         http.MethodPost,
			path:           "/api/reindex",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/reindex/conference/{slug}",
			method:         http.MethodPost,
			path:           "/api/reindex/conference/test-conf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/reindex/talk/{talkId}",
			method:         http.MethodPost,
			path:           "/api/reindex/talk/test-talk-id",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRegisterRoutes_ProductionMode(t *testing.T) {
	ctx := config.WithConfig(context.Background(), testConfigProduction())
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)
	mux := http.NewServeMux()

	adapter.RegisterRoutes(mux)

	// Health check should still be available
	t.Run("GET /health is available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// API routes should NOT be available in production mode
	apiRoutes := []struct {
		name   string
		method string
		path   string
	}{
		{"POST /api/reindex", http.MethodPost, "/api/reindex"},
		{"POST /api/reindex/conference/{slug}", http.MethodPost, "/api/reindex/conference/test-conf"},
		{"POST /api/reindex/talk/{talkId}", http.MethodPost, "/api/reindex/talk/test-talk-id"},
	}

	for _, tt := range apiRoutes {
		t.Run(tt.name+" is not available", func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestRegisterRoutes_MethodNotAllowed(t *testing.T) {
	ctx := config.WithConfig(context.Background(), testConfigDevelopment())
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)
	mux := http.NewServeMux()

	adapter.RegisterRoutes(mux)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "POST /health should not be allowed",
			method: http.MethodPost,
			path:   "/health",
		},
		{
			name:   "GET /api/reindex should not be allowed",
			method: http.MethodGet,
			path:   "/api/reindex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// Go 1.22+ returns 405 Method Not Allowed for wrong methods
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestRegisterRoutes_NotFound(t *testing.T) {
	ctx := config.WithConfig(context.Background(), testConfigDevelopment())
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)
	mux := http.NewServeMux()

	adapter.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterRoutes_Integration(t *testing.T) {
	ctx := config.WithConfig(context.Background(), testConfigDevelopment())

	// Create a mock indexer that tracks calls
	var reindexAllCalled bool
	var reindexConferenceCalled bool
	var reindexConferenceSlug string

	indexer := &mockIndexer{
		reindexAllFunc: func(ctx context.Context) error {
			reindexAllCalled = true
			return nil
		},
		reindexConferenceFunc: func(ctx context.Context, slug string) error {
			reindexConferenceCalled = true
			reindexConferenceSlug = slug
			return nil
		},
	}

	adapter := New(ctx, indexer)
	mux := http.NewServeMux()
	adapter.RegisterRoutes(mux)

	// Test reindex all
	req := httptest.NewRequest(http.MethodPost, "/api/reindex", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.True(t, reindexAllCalled)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test reindex conference
	req = httptest.NewRequest(http.MethodPost, "/api/reindex/conference/javazone-2024", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.True(t, reindexConferenceCalled)
	assert.Equal(t, "javazone-2024", reindexConferenceSlug)
	assert.Equal(t, http.StatusOK, w.Code)
}
