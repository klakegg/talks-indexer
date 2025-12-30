package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleReindexAll_Success(t *testing.T) {
	// Create adapter with mock indexer
	ctx := testContext()
	indexer := &mockIndexer{
		reindexAllFunc: func(ctx context.Context) error {
			return nil
		},
	}
	adapter := New(ctx, indexer)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/reindex", nil)
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleReindexAll(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "success", response.Status)
	assert.Contains(t, response.Message, "successfully reindexed all conferences")
}

func TestHandleReindexAll_Error(t *testing.T) {
	expectedError := errors.New("indexing failed")

	// Create adapter with mock indexer that returns an error
	ctx := testContext()
	indexer := &mockIndexer{
		reindexAllFunc: func(ctx context.Context) error {
			return expectedError
		},
	}
	adapter := New(ctx, indexer)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/reindex", nil)
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleReindexAll(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "error", response.Status)
	assert.Contains(t, response.Message, "failed to reindex all conferences")
	assert.Contains(t, response.Message, expectedError.Error())
}

func TestHandleReindexConference_Success(t *testing.T) {
	var capturedSlug string

	// Create adapter with mock indexer
	ctx := testContext()
	indexer := &mockIndexer{
		reindexConferenceFunc: func(ctx context.Context, slug string) error {
			capturedSlug = slug
			return nil
		},
	}
	adapter := New(ctx, indexer)

	// Create request with slug path parameter
	req := httptest.NewRequest(http.MethodPost, "/api/reindex/javazone-2024", nil)
	req.SetPathValue("slug", "javazone-2024")
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleReindexConference(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Verify the slug was passed to the indexer
	assert.Equal(t, "javazone-2024", capturedSlug)

	// Parse response body
	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "success", response.Status)
	assert.Contains(t, response.Message, "successfully reindexed conference")
	assert.Contains(t, response.Message, "javazone-2024")
}

func TestHandleReindexConference_MissingSlug(t *testing.T) {
	ctx := testContext()
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)

	// Create request without slug
	req := httptest.NewRequest(http.MethodPost, "/api/reindex/", nil)
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleReindexConference(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Parse response body
	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "error", response.Status)
	assert.Contains(t, response.Message, "conference slug is required")
}

func TestHandleReindexConference_Error(t *testing.T) {
	expectedError := errors.New("conference not found")

	// Create adapter with mock indexer that returns an error
	ctx := testContext()
	indexer := &mockIndexer{
		reindexConferenceFunc: func(ctx context.Context, slug string) error {
			return expectedError
		},
	}
	adapter := New(ctx, indexer)

	// Create request with slug
	req := httptest.NewRequest(http.MethodPost, "/api/reindex/invalid-conf", nil)
	req.SetPathValue("slug", "invalid-conf")
	w := httptest.NewRecorder()

	// Call handler
	adapter.HandleReindexConference(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "error", response.Status)
	assert.Contains(t, response.Message, "failed to reindex conference")
	assert.Contains(t, response.Message, expectedError.Error())
}

func TestWriteSuccessResponse(t *testing.T) {
	ctx := testContext()
	adapter := New(ctx, &mockIndexer{})
	w := httptest.NewRecorder()

	response := ReindexResponse{
		Status:  "success",
		Message: "test message",
	}

	adapter.writeSuccessResponse(w, response)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var decoded ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&decoded)
	require.NoError(t, err)

	assert.Equal(t, "success", decoded.Status)
	assert.Equal(t, "test message", decoded.Message)
}

func TestWriteErrorResponse(t *testing.T) {
	ctx := testContext()
	adapter := New(ctx, &mockIndexer{})
	w := httptest.NewRecorder()

	testError := errors.New("test error")
	adapter.writeErrorResponse(w, "operation failed", testError)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "error", response.Status)
	assert.Contains(t, response.Message, "operation failed")
	assert.Contains(t, response.Message, "test error")
}

func TestWriteErrorResponse_NoError(t *testing.T) {
	ctx := testContext()
	adapter := New(ctx, &mockIndexer{})
	w := httptest.NewRecorder()

	adapter.writeErrorResponse(w, "operation failed", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ReindexResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "error", response.Status)
	assert.Equal(t, "operation failed", response.Message)
}
