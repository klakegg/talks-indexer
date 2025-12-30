package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"
	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/domain"
)

// Client implements the SearchIndex interface for Elasticsearch operations.
type Client struct {
	es     *elasticsearch.Client
	logger *slog.Logger
}

// New creates a new Elasticsearch client, retrieving configuration from context.
func New(ctx context.Context) (*Client, error) {
	appCfg := config.GetConfig(ctx)

	esCfg := elasticsearch.Config{
		Addresses: []string{appCfg.Elasticsearch.URL},
	}

	// Add authentication if credentials are provided
	if appCfg.Elasticsearch.HasCredentials() {
		esCfg.Username = appCfg.Elasticsearch.User
		esCfg.Password = appCfg.Elasticsearch.Password
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Verify connection
	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch connection error: %s - %s", res.Status(), string(body))
	}

	logger := slog.Default().With("component", "elasticsearch")
	logger.Info("connected to elasticsearch", "url", appCfg.Elasticsearch.URL, "authenticated", appCfg.Elasticsearch.HasCredentials())

	return &Client{
		es:     es,
		logger: logger,
	}, nil
}

// NewWithURL creates a new Elasticsearch client with explicit URL and credentials.
// This constructor is primarily intended for testing purposes.
func NewWithURL(elasticsearchURL, username, password string) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{elasticsearchURL},
	}

	// Add authentication if credentials are provided
	if username != "" && password != "" {
		cfg.Username = username
		cfg.Password = password
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Verify connection
	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch connection error: %s - %s", res.Status(), string(body))
	}

	logger := slog.Default().With("component", "elasticsearch")
	logger.Info("connected to elasticsearch", "url", elasticsearchURL, "authenticated", username != "")

	return &Client{
		es:     es,
		logger: logger,
	}, nil
}

// BulkIndex indexes multiple talks into the specified index using the Bulk API.
// Each talk is indexed with its ID as the document ID.
func (c *Client) BulkIndex(ctx context.Context, indexName string, talks []domain.Talk) error {
	if len(talks) == 0 {
		c.logger.Info("no talks to index", "index", indexName)
		return nil
	}

	var buf bytes.Buffer

	// Build bulk request body
	for _, talk := range talks {
		// Action metadata
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
				"_id":    talk.ID,
			},
		}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("failed to marshal bulk metadata for talk %s: %w", talk.ID, err)
		}

		// Document body
		docJSON, err := json.Marshal(talk)
		if err != nil {
			return fmt.Errorf("failed to marshal talk %s: %w", talk.ID, err)
		}

		// Write to buffer (each line must be newline-delimited)
		buf.Write(metaJSON)
		buf.WriteByte('\n')
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	// Execute bulk request
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(buf.Bytes()),
		Refresh: "true", // Make documents immediately available for search
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to execute bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bulk index error: %s - %s", res.Status(), string(body))
	}

	// Parse response to check for errors
	var bulkResponse struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
		return fmt.Errorf("failed to parse bulk response: %w", err)
	}

	if bulkResponse.Errors {
		// Collect error details
		var errorDetails []string
		for _, item := range bulkResponse.Items {
			for action, details := range item {
				if details.Status >= 400 {
					errorDetails = append(errorDetails, fmt.Sprintf(
						"%s failed for doc %s (status %d): %s - %s",
						action, details.ID, details.Status, details.Error.Type, details.Error.Reason,
					))
				}
			}
		}
		return fmt.Errorf("bulk index had errors: %s", strings.Join(errorDetails, "; "))
	}

	c.logger.Info("bulk indexed talks", "index", indexName, "count", len(talks))
	return nil
}

// DeleteIndex removes an index from Elasticsearch.
func (c *Client) DeleteIndex(ctx context.Context, indexName string) error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to delete index %s: %w", indexName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		// 404 is acceptable - index already doesn't exist
		if res.StatusCode == http.StatusNotFound {
			c.logger.Info("index does not exist (already deleted)", "index", indexName)
			return nil
		}

		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("delete index error: %s - %s", res.Status(), string(body))
	}

	c.logger.Info("deleted index", "index", indexName)
	return nil
}

// CreateIndex creates a new index with the specified mapping.
func (c *Client) CreateIndex(ctx context.Context, indexName string, mapping string) error {
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", indexName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("create index error: %s - %s", res.Status(), string(body))
	}

	c.logger.Info("created index", "index", indexName)
	return nil
}

// IndexExists checks if an index exists in Elasticsearch.
func (c *Client) IndexExists(ctx context.Context, indexName string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return false, fmt.Errorf("failed to check if index exists %s: %w", indexName, err)
	}
	defer res.Body.Close()

	// 200 = exists, 404 = does not exist
	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// Any other status is an error
	body, _ := io.ReadAll(res.Body)
	return false, fmt.Errorf("index exists check error: %s - %s", res.Status(), string(body))
}
