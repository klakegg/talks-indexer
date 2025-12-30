package api

import (
	"context"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/ports"
)

// Adapter holds the API adapter dependencies
type Adapter struct {
	indexer ports.Indexer
	cfg     *config.Config
}

// New creates a new API adapter
func New(ctx context.Context, indexer ports.Indexer) *Adapter {
	return &Adapter{
		indexer: indexer,
		cfg:     config.GetConfig(ctx),
	}
}
