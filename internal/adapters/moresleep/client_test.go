package moresleep

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a test config with the given moresleep URL
func testConfig(moresleepURL string) *config.Config {
	return &config.Config{
		Moresleep: config.MoresleepConfig{
			URL:      moresleepURL,
			User:     "",
			Password: "",
		},
	}
}

// testConfigWithAuth creates a test config with moresleep URL and credentials
func testConfigWithAuth(moresleepURL, user, password string) *config.Config {
	return &config.Config{
		Moresleep: config.MoresleepConfig{
			URL:      moresleepURL,
			User:     user,
			Password: password,
		},
	}
}

func TestNew(t *testing.T) {
	t.Run("successful creation with context config", func(t *testing.T) {
		cfg := testConfig("https://api.example.com")
		ctx := config.WithConfig(context.Background(), cfg)

		client, err := New(ctx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://api.example.com", client.baseURL)
		assert.Equal(t, "", client.username)
		assert.Equal(t, "", client.password)
	})

	t.Run("successful creation with auth config", func(t *testing.T) {
		cfg := testConfigWithAuth("https://api.example.com", "user", "pass")
		ctx := config.WithConfig(context.Background(), cfg)

		client, err := New(ctx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://api.example.com", client.baseURL)
		assert.Equal(t, "user", client.username)
		assert.Equal(t, "pass", client.password)
	})

	t.Run("panics when config not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			New(ctx)
		})
	})
}

func TestClient_GetConferences(t *testing.T) {
	t.Run("successful fetch with wrapped response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/data/conference", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			response := ConferencesAPIResponse{
				Conferences: []ConferenceResponse{
					{
						ID:   "conf-1",
						Name: "JavaZone 2024",
						Slug: "javazone2024",
					},
					{
						ID:   "conf-2",
						Name: "JavaZone 2023",
						Slug: "javazone2023",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		conferences, err := client.GetConferences(context.Background())

		require.NoError(t, err)
		assert.Len(t, conferences, 2)
		assert.Equal(t, "conf-1", conferences[0].ID)
		assert.Equal(t, "JavaZone 2024", conferences[0].Name)
		assert.Equal(t, "javazone2024", conferences[0].Slug)
	})

	t.Run("successful fetch with direct array response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := []ConferenceResponse{
				{
					ID:   "conf-1",
					Name: "JavaZone 2024",
					Slug: "javazone2024",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		conferences, err := client.GetConferences(context.Background())

		require.NoError(t, err)
		assert.Len(t, conferences, 1)
		assert.Equal(t, "conf-1", conferences[0].ID)
	})

	t.Run("with basic auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "testuser", username)
			assert.Equal(t, "testpass", password)

			response := ConferencesAPIResponse{
				Conferences: []ConferenceResponse{
					{ID: "conf-1", Name: "Test", Slug: "test"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "testuser", "testpass", &http.Client{})
		conferences, err := client.GetConferences(context.Background())

		require.NoError(t, err)
		assert.Len(t, conferences, 1)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		conferences, err := client.GetConferences(context.Background())

		require.Error(t, err)
		assert.Nil(t, conferences)
		assert.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		conferences, err := client.GetConferences(context.Background())

		require.Error(t, err)
		assert.Nil(t, conferences)
		assert.Contains(t, err.Error(), "failed to unmarshal conferences")
	})
}

func TestClient_GetTalks(t *testing.T) {
	now := time.Now()
	startTime := now.Add(1 * time.Hour)
	endTime := now.Add(2 * time.Hour)

	t.Run("successful fetch with wrapped response", func(t *testing.T) {
		var conferenceCall, sessionCall bool

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			if r.URL.Path == "/data/conference" {
				conferenceCall = true
				response := ConferencesAPIResponse{
					Conferences: []ConferenceResponse{
						{
							ID:   "conf-1",
							Name: "JavaZone 2024",
							Slug: "javazone2024",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.URL.Path == "/data/conference/conf-1/session" {
				sessionCall = true
				response := SessionsAPIResponse{
					Sessions: []SessionResponse{
						{
							ID:           "talk-1",
							ConferenceID: "conf-1",
							Status:       "APPROVED",
							PostedBy:     "speaker@example.com",
							Data: map[string]DataValue{
								"title":            {Value: "Introduction to Go", PrivateData: false},
								"abstract":         {Value: "A comprehensive introduction to Go programming", PrivateData: false},
								"intendedAudience": {Value: "Beginners", PrivateData: false},
								"language":         {Value: "English", PrivateData: false},
								"format":           {Value: "Presentation", PrivateData: false},
								"level":            {Value: "Beginner", PrivateData: false},
								"keywords":         {Value: []interface{}{"go", "programming", "tutorial"}, PrivateData: false},
								"room":             {Value: "Room A", PrivateData: false},
								"startTime":        {Value: startTime.Format(time.RFC3339), PrivateData: false},
								"endTime":          {Value: endTime.Format(time.RFC3339), PrivateData: false},
							},
							Speakers: []SpeakerResponse{
								{
									ID:    "speaker-1",
									Name:  "John Doe",
									Email: "john@example.com",
									Data: map[string]DataValue{
										"bio":        {Value: "Expert Go developer", PrivateData: false},
										"twitter":    {Value: "@johndoe", PrivateData: false},
										"pictureUrl": {Value: "https://example.com/john.jpg", PrivateData: false},
									},
								},
							},
							Created:     FlexibleTime{Time: now},
							LastUpdated: FlexibleTime{Time: now},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		talks, err := client.GetTalks(context.Background(), "conf-1")

		require.NoError(t, err)
		assert.True(t, conferenceCall, "conference endpoint should have been called")
		assert.True(t, sessionCall, "session endpoint should have been called")
		assert.Len(t, talks, 1)

		talk := talks[0]
		assert.Equal(t, "talk-1", talk.ID)
		assert.Equal(t, "conf-1", talk.ConferenceID)
		assert.Equal(t, "javazone2024", talk.ConferenceSlug)
		assert.Equal(t, "Introduction to Go", talk.Data["title"])
		assert.Equal(t, "A comprehensive introduction to Go programming", talk.Data["abstract"])
		assert.Equal(t, "Beginners", talk.Data["intendedAudience"])
		assert.Equal(t, "English", talk.Data["language"])
		assert.Equal(t, "Presentation", talk.Data["format"])
		assert.Equal(t, "Beginner", talk.Data["level"])
		assert.Equal(t, []interface{}{"go", "programming", "tutorial"}, talk.Data["keywords"])
		assert.Equal(t, "APPROVED", talk.Status)
		assert.Equal(t, "Room A", talk.Data["room"])
		assert.Equal(t, "speaker@example.com", talk.PrivateData["postedBy"])

		require.Len(t, talk.Speakers, 1)
		speaker := talk.Speakers[0]
		assert.Equal(t, "speaker-1", speaker.ID)
		assert.Equal(t, "John Doe", speaker.Name)
		assert.Equal(t, "Expert Go developer", speaker.Data["bio"])
		assert.Equal(t, "@johndoe", speaker.Data["twitter"])
		assert.Equal(t, "https://example.com/john.jpg", speaker.Data["pictureUrl"])
	})

	t.Run("successful fetch with direct array response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/data/conference" {
				response := []ConferenceResponse{
					{ID: "conf-1", Name: "Test", Slug: "test"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.URL.Path == "/data/conference/conf-1/session" {
				response := []SessionResponse{
					{
						ID:           "talk-1",
						ConferenceID: "conf-1",
						Status:       "SUBMITTED",
						PostedBy:     "test@example.com",
						Data: map[string]DataValue{
							"title": {Value: "Test Talk", PrivateData: false},
						},
						Speakers:    []SpeakerResponse{},
						Created:     FlexibleTime{Time: now},
						LastUpdated: FlexibleTime{Time: now},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		talks, err := client.GetTalks(context.Background(), "conf-1")

		require.NoError(t, err)
		assert.Len(t, talks, 1)
		assert.Equal(t, "Test Talk", talks[0].Data["title"])
	})

	t.Run("conference not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/data/conference" {
				response := ConferencesAPIResponse{
					Conferences: []ConferenceResponse{},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.URL.Path == "/data/conference/nonexistent/session" {
				response := SessionsAPIResponse{
					Sessions: []SessionResponse{
						{
							ID:           "talk-1",
							ConferenceID: "nonexistent",
							Status:       "APPROVED",
							PostedBy:     "test@example.com",
							Data:         map[string]DataValue{},
							Speakers:     []SpeakerResponse{},
							Created:      FlexibleTime{Time: now},
							LastUpdated:  FlexibleTime{Time: now},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		talks, err := client.GetTalks(context.Background(), "nonexistent")

		// Should still succeed but with empty slug
		require.NoError(t, err)
		assert.Len(t, talks, 1)
		assert.Equal(t, "", talks[0].ConferenceSlug)
	})

	t.Run("server error on sessions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		talks, err := client.GetTalks(context.Background(), "conf-1")

		require.Error(t, err)
		assert.Nil(t, talks)
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/data/conference" {
				response := ConferencesAPIResponse{
					Conferences: []ConferenceResponse{
						{ID: "conf-1", Name: "Test", Slug: "test"},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewWithHTTPClient(server.URL, "", "", &http.Client{})
		talks, err := client.GetTalks(context.Background(), "conf-1")

		require.Error(t, err)
		assert.Nil(t, talks)
		assert.Contains(t, err.Error(), "failed to unmarshal sessions")
	})
}

func TestClient_NewWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	client := NewWithHTTPClient("https://example.com", "user", "pass", customClient)

	assert.NotNil(t, client)
	assert.Equal(t, "https://example.com", client.baseURL)
	assert.Equal(t, "user", client.username)
	assert.Equal(t, "pass", client.password)
	assert.Equal(t, customClient, client.httpClient)
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// This test ensures that Client implements the TalkSource interface
	var _ domain.Conference // Ensure domain types are imported

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ConferencesAPIResponse{
			Conferences: []ConferenceResponse{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewWithHTTPClient(server.URL, "", "", &http.Client{})

	// Test that we can use the client through the interface
	ctx := context.Background()
	conferences, err := client.GetConferences(ctx)
	require.NoError(t, err)
	assert.NotNil(t, conferences)
}
