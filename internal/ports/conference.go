package ports

import (
	"context"

	"github.com/javaBin/talks-indexer/internal/domain"
)

// ConferenceProvider defines the interface for fetching conferences.
// This is a subset of TalkSource, following the Interface Segregation Principle.
// Any TalkSource implementation will automatically satisfy this interface.
type ConferenceProvider interface {
	// GetConferences retrieves all available conferences
	GetConferences(ctx context.Context) ([]domain.Conference, error)
}
