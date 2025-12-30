package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleHealth(t *testing.T) {
	// Create adapter with mock indexer
	ctx := testContext()
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleHealth(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
}

func TestHandleHealth_ContentType(t *testing.T) {
	ctx := testContext()
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	adapter.HandleHealth(w, req)

	contentType := w.Header().Get("Content-Type")
	assert.Equal(t, "application/json", contentType)
}

func TestHandleHealth_StatusCode(t *testing.T) {
	ctx := testContext()
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	adapter.HandleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
