package config

import (
	"context"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// contextKey is a private type used for context keys to avoid collisions
type contextKey string

const configKey contextKey = "config"

// Mode represents the application running mode
type Mode string

const (
	ModeProduction  Mode = "production"
	ModeDevelopment Mode = "development"
)

// IsDevelopment returns true if the mode is development
func (m Mode) IsDevelopment() bool {
	return m == ModeDevelopment
}

// Config holds all application configuration loaded from environment variables
type Config struct {
	Mode              Mode   `env:"MODE" envDefault:"production"`
	Port              int    `env:"PORT" envDefault:"8080"`
	MoresleepURL      string `env:"MORESLEEP_URL" envDefault:"http://localhost:8082"`
	MoresleepUser     string `env:"MORESLEEP_USER"`
	MoresleepPassword string `env:"MORESLEEP_PASSWORD"`
	ElasticsearchURL  string `env:"ELASTICSEARCH_URL" envDefault:"http://localhost:9200"`
	PrivateIndex      string `env:"PRIVATE_INDEX" envDefault:"javazone_private"`
	PublicIndex       string `env:"PUBLIC_INDEX" envDefault:"javazone_public"`

	// OIDC Configuration (only used in production mode)
	OIDCIssuerURL    string `env:"OIDC_ISSUER_URL"`
	OIDCClientID     string `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret string `env:"OIDC_CLIENT_SECRET"`
	OIDCRedirectURL  string `env:"OIDC_REDIRECT_URL"`
}

// IsOIDCConfigured returns true if OIDC is fully configured
func (c *Config) IsOIDCConfigured() bool {
	return c.OIDCIssuerURL != "" &&
		c.OIDCClientID != "" &&
		c.OIDCClientSecret != ""
}

// Load reads configuration from environment variables and optionally from a .env file.
// It returns a pointer to the Config struct or an error if parsing fails.
func Load() (*Config, error) {
	// Try to load .env file, but ignore error if it doesn't exist
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return cfg, nil
}

// WithConfig returns a new context with the provided Config attached
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

// GetConfig retrieves the Config from the context.
// It panics if the config is not found in the context.
func GetConfig(ctx context.Context) *Config {
	cfg, ok := ctx.Value(configKey).(*Config)
	if !ok {
		panic("config not found in context")
	}
	return cfg
}

// MustLoad loads the configuration and panics if it fails.
// This is useful for initialization in main() where we want to fail fast.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
