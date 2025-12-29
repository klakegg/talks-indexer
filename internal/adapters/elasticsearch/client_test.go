package elasticsearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/javaBin/talks-indexer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		// Create mock Elasticsearch server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				// Root endpoint returns cluster info
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"name":         "test-cluster",
					"cluster_name": "elasticsearch",
					"version": map[string]interface{}{
						"number": "8.0.0",
					},
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NotNil(t, client.es)
		assert.NotNil(t, client.logger)
	})

	t.Run("connection failure", func(t *testing.T) {
		client, err := New("http://invalid-host:9999")
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("error response from server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client, err := New(server.URL)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "elasticsearch connection error")
	})
}

func TestClient_CreateIndex(t *testing.T) {
	t.Run("successful index creation", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && r.URL.Path == "/test-index" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"acknowledged":        true,
					"shards_acknowledged": true,
					"index":               "test-index",
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.CreateIndex(context.Background(), "test-index", TalkPrivateIndexMapping)
		assert.NoError(t, err)
	})

	t.Run("index creation error", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "PUT" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid mapping"}`))
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.CreateIndex(context.Background(), "test-index", "invalid-json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create index error")
	})
}

func TestClient_DeleteIndex(t *testing.T) {
	t.Run("successful deletion", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" && r.URL.Path == "/test-index" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"acknowledged": true,
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.DeleteIndex(context.Background(), "test-index")
		assert.NoError(t, err)
	})

	t.Run("index not found (acceptable)", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"index_not_found_exception"}`))
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.DeleteIndex(context.Background(), "test-index")
		assert.NoError(t, err) // Should not error on 404
	})

	t.Run("other error", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.DeleteIndex(context.Background(), "test-index")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete index error")
	})
}

func TestClient_IndexExists(t *testing.T) {
	t.Run("index exists", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		exists, err := client.IndexExists(context.Background(), "test-index")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("index does not exist", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		exists, err := client.IndexExists(context.Background(), "test-index")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("server error", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" && r.URL.Path == "/test-index" {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		exists, err := client.IndexExists(context.Background(), "test-index")
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "index exists check error")
	})
}

func TestClient_BulkIndex(t *testing.T) {
	t.Run("successful bulk indexing", func(t *testing.T) {
		var receivedBody string
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/_bulk" {
				bodyBytes, _ := io.ReadAll(r.Body)
				receivedBody = string(bodyBytes)

				// Return successful bulk response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"took":   5,
					"errors": false,
					"items": []map[string]interface{}{
						{
							"index": map[string]interface{}{
								"_id":      "talk-1",
								"status":   201,
								"result":   "created",
								"_version": 1,
							},
						},
						{
							"index": map[string]interface{}{
								"_id":      "talk-2",
								"status":   201,
								"result":   "created",
								"_version": 1,
							},
						},
					},
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		talks := createTestTalks(2)
		err = client.BulkIndex(context.Background(), "test-index", talks)
		assert.NoError(t, err)

		// Verify bulk request format
		assert.Contains(t, receivedBody, `"_index":"test-index"`)
		assert.Contains(t, receivedBody, `"_id":"talk-1"`)
		assert.Contains(t, receivedBody, `"_id":"talk-2"`)
		assert.Contains(t, receivedBody, `"title":"Test Talk 1"`)
	})

	t.Run("empty talks array", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		err = client.BulkIndex(context.Background(), "test-index", []domain.Talk{})
		assert.NoError(t, err) // Should not error for empty array
	})

	t.Run("bulk indexing with errors", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/_bulk" {
				// Return bulk response with errors
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"took":   5,
					"errors": true,
					"items": []map[string]interface{}{
						{
							"index": map[string]interface{}{
								"_id":    "talk-1",
								"status": 201,
								"result": "created",
							},
						},
						{
							"index": map[string]interface{}{
								"_id":    "talk-2",
								"status": 400,
								"error": map[string]interface{}{
									"type":   "mapper_parsing_exception",
									"reason": "failed to parse field",
								},
							},
						},
					},
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		talks := createTestTalks(2)
		err = client.BulkIndex(context.Background(), "test-index", talks)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bulk index had errors")
		assert.Contains(t, err.Error(), "mapper_parsing_exception")
		assert.Contains(t, err.Error(), "talk-2")
	})

	t.Run("http error response", func(t *testing.T) {
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/_bulk" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"server error"}`))
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		talks := createTestTalks(1)
		err = client.BulkIndex(context.Background(), "test-index", talks)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bulk index error")
	})
}

func TestClient_BulkIndexFormat(t *testing.T) {
	t.Run("verify bulk request format", func(t *testing.T) {
		var receivedLines []string
		server := createMockESServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/_bulk" {
				bodyBytes, _ := io.ReadAll(r.Body)
				receivedLines = strings.Split(string(bodyBytes), "\n")

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": false,
					"items": []map[string]interface{}{
						{"index": map[string]interface{}{"_id": "talk-1", "status": 201}},
					},
				})
			}
		}))
		defer server.Close()

		client, err := New(server.URL)
		require.NoError(t, err)

		talks := createTestTalks(1)
		err = client.BulkIndex(context.Background(), "test-index", talks)
		require.NoError(t, err)

		// Bulk API format: action_and_meta_data\n + optional_source\n
		// For one document, we expect 2 lines + trailing newline = 3 elements (last is empty)
		require.Len(t, receivedLines, 3, "Expected 3 lines (action, source, empty)")

		// Parse action line
		var action map[string]interface{}
		err = json.Unmarshal([]byte(receivedLines[0]), &action)
		require.NoError(t, err)
		require.Contains(t, action, "index")
		indexAction := action["index"].(map[string]interface{})
		assert.Equal(t, "test-index", indexAction["_index"])
		assert.Equal(t, "talk-1", indexAction["_id"])

		// Parse document line
		var doc map[string]interface{}
		err = json.Unmarshal([]byte(receivedLines[1]), &doc)
		require.NoError(t, err)
		assert.Equal(t, "talk-1", doc["id"])
		data := doc["data"].(map[string]interface{})
		assert.Equal(t, "Test Talk 1", data["title"])
	})
}

// Helper function to create a mock Elasticsearch server
func createMockESServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add Elasticsearch product header to all responses
		w.Header().Set("X-Elastic-Product", "Elasticsearch")

		// Handle info endpoint for client initialization
		if r.Method == "GET" && r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": map[string]interface{}{"number": "8.0.0"},
			})
			return
		}

		// Delegate to custom handler
		handler(w, r)
	}))
}

// Helper function to create test talks
func createTestTalks(count int) []domain.Talk {
	talks := make([]domain.Talk, count)
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
		id := i + 1
		startTime := baseTime.Add(time.Duration(i) * time.Hour)
		endTime := startTime.Add(45 * time.Minute)

		talks[i] = domain.Talk{
			ID:             "talk-" + string(rune('0'+id)),
			ConferenceID:   "conf-123",
			ConferenceSlug: "javazone",
			Status:         "approved",
			Data: map[string]interface{}{
				"title":            "Test Talk " + string(rune('0'+id)),
				"abstract":         "This is a test abstract for talk " + string(rune('0'+id)),
				"intendedAudience": "Developers",
				"language":         "en",
				"format":           "presentation",
				"level":            "intermediate",
				"keywords":         []string{"java", "testing"},
				"room":             "Room A",
				"startTime":        startTime.Format(time.RFC3339),
				"endTime":          endTime.Format(time.RFC3339),
			},
			PrivateData: map[string]interface{}{
				"postedBy": "submitter@example.com",
			},
			Speakers: domain.Speakers{
				{
					ID:   "speaker-1",
					Name: "John Doe",
					Data: map[string]interface{}{
						"bio":        "Software Developer",
						"twitter":    "@johndoe",
						"pictureUrl": "https://example.com/john.jpg",
					},
				},
			},
			Created:     &baseTime,
			LastUpdated: &baseTime,
		}
	}

	return talks
}
