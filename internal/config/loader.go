package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

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
