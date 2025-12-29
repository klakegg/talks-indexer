package ports

import "context"

// Indexer defines the interface for indexing operations.
// This is implemented by the app layer IndexerService.
type Indexer interface {
	// ReindexAll triggers a full reindex of all conferences
	ReindexAll(ctx context.Context) error

	// ReindexConference reindexes a specific conference by its slug
	ReindexConference(ctx context.Context, slug string) error

	// ReindexTalk reindexes a specific talk by its ID
	ReindexTalk(ctx context.Context, talkID string) error
}
