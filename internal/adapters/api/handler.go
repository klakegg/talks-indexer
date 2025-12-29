package api

import "github.com/javaBin/talks-indexer/internal/ports"

// Handler holds the HTTP handler dependencies
type Handler struct {
	indexer ports.Indexer
}

// NewHandler creates a new HTTP handler with the provided indexer service
func NewHandler(indexer ports.Indexer) *Handler {
	return &Handler{
		indexer: indexer,
	}
}
