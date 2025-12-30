package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
		wantErr  bool
	}{
		{
			name:    "load with defaults",
			envVars: map[string]string{},
			expected: &Config{
				Http: HttpConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				Moresleep: MoresleepConfig{
					URL:      "http://localhost:8082",
					User:     "",
					Password: "",
				},
				Elasticsearch: ElasticsearchConfig{
					URL: "http://localhost:9200",
				},
				Index: IndexConfig{
					Private: "javazone_private",
					Public:  "javazone_public",
				},
			},
			wantErr: false,
		},
		{
			name: "load with custom values",
			envVars: map[string]string{
				"HTTP_HOST":          "127.0.0.1",
				"HTTP_PORT":          "9090",
				"MORESLEEP_URL":      "https://api.example.com",
				"MORESLEEP_USER":     "testuser",
				"MORESLEEP_PASSWORD": "testpass",
				"ELASTICSEARCH_URL":  "https://es.example.com:9200",
				"PRIVATE_INDEX":      "custom_private",
				"PUBLIC_INDEX":       "custom_public",
			},
			expected: &Config{
				Http: HttpConfig{
					Host: "127.0.0.1",
					Port: 9090,
				},
				Moresleep: MoresleepConfig{
					URL:      "https://api.example.com",
					User:     "testuser",
					Password: "testpass",
				},
				Elasticsearch: ElasticsearchConfig{
					URL: "https://es.example.com:9200",
				},
				Index: IndexConfig{
					Private: "custom_private",
					Public:  "custom_public",
				},
			},
			wantErr: false,
		},
		{
			name: "load with partial custom values",
			envVars: map[string]string{
				"HTTP_PORT":      "3000",
				"MORESLEEP_USER": "admin",
			},
			expected: &Config{
				Http: HttpConfig{
					Host: "0.0.0.0",
					Port: 3000,
				},
				Moresleep: MoresleepConfig{
					URL:      "http://localhost:8082",
					User:     "admin",
					Password: "",
				},
				Elasticsearch: ElasticsearchConfig{
					URL: "http://localhost:9200",
				},
				Index: IndexConfig{
					Private: "javazone_private",
					Public:  "javazone_public",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port value",
			envVars: map[string]string{
				"HTTP_PORT": "invalid",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			clearConfigEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearConfigEnv()

			cfg, err := Load()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.Equal(t, tt.expected.Http.Host, cfg.Http.Host)
				assert.Equal(t, tt.expected.Http.Port, cfg.Http.Port)
				assert.Equal(t, tt.expected.Moresleep.URL, cfg.Moresleep.URL)
				assert.Equal(t, tt.expected.Moresleep.User, cfg.Moresleep.User)
				assert.Equal(t, tt.expected.Moresleep.Password, cfg.Moresleep.Password)
				assert.Equal(t, tt.expected.Elasticsearch.URL, cfg.Elasticsearch.URL)
				assert.Equal(t, tt.expected.Index.Private, cfg.Index.Private)
				assert.Equal(t, tt.expected.Index.Public, cfg.Index.Public)
			}
		})
	}
}

func TestHttpConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		config   HttpConfig
		expected string
	}{
		{
			name:     "default values",
			config:   HttpConfig{Host: "0.0.0.0", Port: 8080},
			expected: "0.0.0.0:8080",
		},
		{
			name:     "localhost",
			config:   HttpConfig{Host: "127.0.0.1", Port: 3000},
			expected: "127.0.0.1:3000",
		},
		{
			name:     "custom host and port",
			config:   HttpConfig{Host: "192.168.1.1", Port: 9090},
			expected: "192.168.1.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.Addr())
		})
	}
}

func TestWithConfig(t *testing.T) {
	cfg := &Config{
		Http: HttpConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Moresleep: MoresleepConfig{
			URL:      "http://localhost:8082",
			User:     "testuser",
			Password: "testpass",
		},
		Elasticsearch: ElasticsearchConfig{
			URL: "http://localhost:9200",
		},
		Index: IndexConfig{
			Private: "javazone_private",
			Public:  "javazone_public",
		},
	}

	ctx := context.Background()
	ctxWithConfig := WithConfig(ctx, cfg)

	assert.NotNil(t, ctxWithConfig)
	assert.NotEqual(t, ctx, ctxWithConfig)
}

func TestGetConfig(t *testing.T) {
	t.Run("get config from context", func(t *testing.T) {
		cfg := &Config{
			Http: HttpConfig{
				Host: "127.0.0.1",
				Port: 9090,
			},
			Moresleep: MoresleepConfig{
				URL:      "https://api.example.com",
				User:     "user",
				Password: "pass",
			},
			Elasticsearch: ElasticsearchConfig{
				URL: "https://es.example.com:9200",
			},
			Index: IndexConfig{
				Private: "private",
				Public:  "public",
			},
		}

		ctx := WithConfig(context.Background(), cfg)
		retrievedCfg := GetConfig(ctx)

		require.NotNil(t, retrievedCfg)
		assert.Equal(t, cfg.Http.Host, retrievedCfg.Http.Host)
		assert.Equal(t, cfg.Http.Port, retrievedCfg.Http.Port)
		assert.Equal(t, cfg.Moresleep.URL, retrievedCfg.Moresleep.URL)
		assert.Equal(t, cfg.Moresleep.User, retrievedCfg.Moresleep.User)
		assert.Equal(t, cfg.Moresleep.Password, retrievedCfg.Moresleep.Password)
		assert.Equal(t, cfg.Elasticsearch.URL, retrievedCfg.Elasticsearch.URL)
		assert.Equal(t, cfg.Index.Private, retrievedCfg.Index.Private)
		assert.Equal(t, cfg.Index.Public, retrievedCfg.Index.Public)
	})

	t.Run("panic when config not in context", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			GetConfig(ctx)
		})
	})
}

func TestOIDCConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		oidc     OIDCConfig
		expected bool
	}{
		{
			name:     "empty config",
			oidc:     OIDCConfig{},
			expected: false,
		},
		{
			name: "partial config - missing client secret",
			oidc: OIDCConfig{
				IssuerURL: "https://issuer.example.com",
				ClientID:  "client-id",
			},
			expected: false,
		},
		{
			name: "fully configured",
			oidc: OIDCConfig{
				IssuerURL:    "https://issuer.example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			expected: true,
		},
		{
			name: "fully configured with redirect URL",
			oidc: OIDCConfig{
				IssuerURL:    "https://issuer.example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example.com/callback",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.oidc.IsConfigured())
		})
	}
}

func TestMustLoad(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		clearConfigEnv()
		defer clearConfigEnv()

		os.Setenv("HTTP_PORT", "8080")

		cfg := MustLoad()
		require.NotNil(t, cfg)
		assert.Equal(t, 8080, cfg.Http.Port)
	})
}

// clearConfigEnv removes all config-related environment variables
func clearConfigEnv() {
	os.Unsetenv("MODE")
	os.Unsetenv("HTTP_HOST")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("MORESLEEP_URL")
	os.Unsetenv("MORESLEEP_USER")
	os.Unsetenv("MORESLEEP_PASSWORD")
	os.Unsetenv("ELASTICSEARCH_URL")
	os.Unsetenv("ELASTICSEARCH_USER")
	os.Unsetenv("ELASTICSEARCH_PASSWORD")
	os.Unsetenv("PRIVATE_INDEX")
	os.Unsetenv("PUBLIC_INDEX")
	os.Unsetenv("OIDC_ISSUER_URL")
	os.Unsetenv("OIDC_CLIENT_ID")
	os.Unsetenv("OIDC_CLIENT_SECRET")
	os.Unsetenv("OIDC_REDIRECT_URL")
}
