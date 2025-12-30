package moresleep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/domain"
)

// Client implements the TalkSource interface for the moresleep API
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	logger     *slog.Logger
}

// New creates a new moresleep Client, retrieving configuration from context
// If username and password are configured, Basic Auth will be used for all requests
func New(ctx context.Context) (*Client, error) {
	cfg := config.GetConfig(ctx)
	return &Client{
		baseURL:  cfg.Moresleep.URL,
		username: cfg.Moresleep.User,
		password: cfg.Moresleep.Password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: slog.Default(),
	}, nil
}

// NewWithHTTPClient creates a new moresleep Client with a custom HTTP client.
// This constructor is primarily intended for testing purposes.
func NewWithHTTPClient(baseURL, username, password string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: httpClient,
		logger:     slog.Default(),
	}
}

// SetLogger sets a custom logger for the client
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// doRequest performs an HTTP request with optional Basic Auth
func (c *Client) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Basic Auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	req.Header.Set("Accept", "application/json")

	c.logger.DebugContext(ctx, "Making HTTP request",
		"method", method,
		"url", url,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, "HTTP request failed",
			"status", resp.StatusCode,
			"url", url,
			"body", string(body),
		)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	c.logger.DebugContext(ctx, "HTTP request successful",
		"status", resp.StatusCode,
		"url", url,
	)

	return body, nil
}

// GetConferences retrieves all available conferences from the moresleep API
func (c *Client) GetConferences(ctx context.Context) ([]domain.Conference, error) {
	c.logger.InfoContext(ctx, "Fetching conferences from moresleep API")

	body, err := c.doRequest(ctx, http.MethodGet, "/data/conference")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conferences: %w", err)
	}

	var response ConferencesAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// Try to parse as direct array for backward compatibility
		var conferences []ConferenceResponse
		if err := json.Unmarshal(body, &conferences); err != nil {
			c.logger.ErrorContext(ctx, "Failed to unmarshal conferences response",
				"error", err,
				"body", string(body),
			)
			return nil, fmt.Errorf("failed to unmarshal conferences: %w", err)
		}
		response.Conferences = conferences
	}

	conferences := MapConferences(response.Conferences)

	c.logger.InfoContext(ctx, "Successfully fetched conferences",
		"count", len(conferences),
	)

	return conferences, nil
}

// GetTalks retrieves all talks for a specific conference from the moresleep API
func (c *Client) GetTalks(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
	c.logger.InfoContext(ctx, "Fetching talks from moresleep API",
		"conferenceID", conferenceID,
	)

	path := fmt.Sprintf("/data/conference/%s/session", conferenceID)
	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch talks for conference %s: %w", conferenceID, err)
	}

	var response SessionsAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// Try to parse as direct array for backward compatibility
		var sessions []SessionResponse
		if err := json.Unmarshal(body, &sessions); err != nil {
			c.logger.ErrorContext(ctx, "Failed to unmarshal sessions response",
				"error", err,
				"conferenceID", conferenceID,
				"body", string(body),
			)
			return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
		}
		response.Sessions = sessions
	}

	// We need to get the conference slug and name for mapping
	// First, fetch the conference to get its details
	conferences, err := c.GetConferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conferences to get details: %w", err)
	}

	var conferenceSlug, conferenceName string
	for _, conf := range conferences {
		if conf.ID == conferenceID {
			conferenceSlug = conf.Slug
			conferenceName = conf.Name
			break
		}
	}

	if conferenceSlug == "" {
		c.logger.WarnContext(ctx, "Conference not found, using empty strings",
			"conferenceID", conferenceID,
		)
	}

	talks := MapTalks(response.Sessions, conferenceSlug, conferenceName)

	c.logger.InfoContext(ctx, "Successfully fetched talks",
		"conferenceID", conferenceID,
		"count", len(talks),
	)

	return talks, nil
}

// GetTalk retrieves a single talk by its ID from the moresleep API
func (c *Client) GetTalk(ctx context.Context, talkID string) (*domain.Talk, error) {
	c.logger.InfoContext(ctx, "Fetching talk from moresleep API",
		"talkID", talkID,
	)

	path := fmt.Sprintf("/data/session/%s", talkID)
	body, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch talk %s: %w", talkID, err)
	}

	var session SessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		c.logger.ErrorContext(ctx, "Failed to unmarshal session response",
			"error", err,
			"talkID", talkID,
			"body", string(body),
		)
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// We need to get the conference slug and name for mapping
	conferences, err := c.GetConferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conferences to get details: %w", err)
	}

	var conferenceSlug, conferenceName string
	for _, conf := range conferences {
		if conf.ID == session.ConferenceID {
			conferenceSlug = conf.Slug
			conferenceName = conf.Name
			break
		}
	}

	if conferenceSlug == "" {
		c.logger.WarnContext(ctx, "Conference not found, using empty strings",
			"conferenceID", session.ConferenceID,
		)
	}

	talk := MapTalk(session, conferenceSlug, conferenceName)

	c.logger.InfoContext(ctx, "Successfully fetched talk",
		"talkID", talkID,
		"conferenceSlug", conferenceSlug,
	)

	return &talk, nil
}
