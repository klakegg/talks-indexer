package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes(t *testing.T) {
	indexer := &mockIndexer{}
	handler := NewHandler(indexer)
	mux := http.NewServeMux()

	RegisterRoutes(mux, handler)

	// Test that routes are registered by making requests
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

func TestRegisterRoutes_MethodNotAllowed(t *testing.T) {
	indexer := &mockIndexer{}
	handler := NewHandler(indexer)
	mux := http.NewServeMux()

	RegisterRoutes(mux, handler)

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
	indexer := &mockIndexer{}
	handler := NewHandler(indexer)
	mux := http.NewServeMux()

	RegisterRoutes(mux, handler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterRoutes_Integration(t *testing.T) {
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

	handler := NewHandler(indexer)
	mux := http.NewServeMux()
	RegisterRoutes(mux, handler)

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
