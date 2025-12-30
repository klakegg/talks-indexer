package config

import "context"

// contextKey is a private type used for context keys to avoid collisions
type contextKey string

const configKey contextKey = "config"

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
