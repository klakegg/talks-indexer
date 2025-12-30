package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/domain"
	"github.com/javaBin/talks-indexer/internal/ports"
)

// IndexerService handles the business logic for indexing talks
type IndexerService struct {
	source              ports.TalkSource
	searchIndex         ports.SearchIndex
	privateIndex        string
	publicIndex         string
	privateIndexMapping string
	publicIndexMapping  string
	logger              *slog.Logger
}

// NewIndexerService creates a new IndexerService, receiving context as first parameter
// to retrieve configuration, along with the required port dependencies.
func NewIndexerService(
	ctx context.Context,
	source ports.TalkSource,
	searchIndex ports.SearchIndex,
	privateIndexMapping string,
	publicIndexMapping string,
) *IndexerService {
	cfg := config.GetConfig(ctx)
	return &IndexerService{
		source:              source,
		searchIndex:         searchIndex,
		privateIndex:        cfg.Index.Private,
		publicIndex:         cfg.Index.Public,
		privateIndexMapping: privateIndexMapping,
		publicIndexMapping:  publicIndexMapping,
		logger:              slog.Default().With("component", "indexer"),
	}
}

// NewIndexerServiceWithConfig creates a new IndexerService with explicit configuration.
// This constructor is primarily intended for testing purposes.
func NewIndexerServiceWithConfig(
	source ports.TalkSource,
	searchIndex ports.SearchIndex,
	privateIndex string,
	publicIndex string,
	privateIndexMapping string,
	publicIndexMapping string,
) *IndexerService {
	return &IndexerService{
		source:              source,
		searchIndex:         searchIndex,
		privateIndex:        privateIndex,
		publicIndex:         publicIndex,
		privateIndexMapping: privateIndexMapping,
		publicIndexMapping:  publicIndexMapping,
		logger:              slog.Default().With("component", "indexer"),
	}
}

// ReindexAll fetches all conferences and their talks, then indexes them
// to both private (all talks) and public (only approved talks) indexes.
func (s *IndexerService) ReindexAll(ctx context.Context) error {
	s.logger.Info("starting full reindex of all conferences")

	// Fetch all conferences
	conferences, err := s.source.GetConferences(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch conferences: %w", err)
	}

	s.logger.Info("fetched conferences", "count", len(conferences))

	// Recreate both indexes
	if err := s.recreateIndex(ctx, s.privateIndex); err != nil {
		return fmt.Errorf("failed to recreate private index: %w", err)
	}
	if err := s.recreateIndex(ctx, s.publicIndex); err != nil {
		return fmt.Errorf("failed to recreate public index: %w", err)
	}

	// Collect all talks from all conferences
	var allTalks []domain.Talk

	for _, conf := range conferences {
		talks, err := s.source.GetTalks(ctx, conf.ID)
		if err != nil {
			s.logger.Error("failed to fetch talks for conference",
				"conferenceID", conf.ID,
				"conferenceName", conf.Name,
				"error", err,
			)
			continue
		}

		s.logger.Info("fetched talks for conference",
			"conferenceID", conf.ID,
			"conferenceName", conf.Name,
			"count", len(talks),
		)

		allTalks = append(allTalks, talks...)
	}

	if len(allTalks) == 0 {
		s.logger.Warn("no talks found to index")
		return nil
	}

	// Index all talks to private index (with privateData merged into data)
	privateTalks := prepareTalksForPrivateIndex(allTalks)
	if err := s.searchIndex.BulkIndex(ctx, s.privateIndex, privateTalks); err != nil {
		return fmt.Errorf("failed to index to private index: %w", err)
	}

	// Filter approved talks for public index (with private data removed)
	publicTalks := filterApprovedTalksForPublic(allTalks)

	s.logger.Info("filtered approved talks for public index",
		"total", len(allTalks),
		"approved", len(publicTalks),
	)

	// Index approved talks to public index
	if err := s.searchIndex.BulkIndex(ctx, s.publicIndex, publicTalks); err != nil {
		return fmt.Errorf("failed to index to public index: %w", err)
	}

	s.logger.Info("full reindex completed successfully",
		"privateCount", len(allTalks),
		"publicCount", len(publicTalks),
	)

	return nil
}

// ReindexConference reindexes talks for a specific conference by its slug.
// It updates both private and public indexes for that conference's talks.
func (s *IndexerService) ReindexConference(ctx context.Context, slug string) error {
	s.logger.Info("starting reindex for conference", "slug", slug)

	// Find the conference by slug
	conferences, err := s.source.GetConferences(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch conferences: %w", err)
	}

	var targetConference *domain.Conference
	for _, conf := range conferences {
		if conf.Slug == slug {
			targetConference = &conf
			break
		}
	}

	if targetConference == nil {
		return fmt.Errorf("conference not found with slug: %s", slug)
	}

	// Fetch talks for this conference
	talks, err := s.source.GetTalks(ctx, targetConference.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch talks for conference %s: %w", slug, err)
	}

	s.logger.Info("fetched talks for conference",
		"slug", slug,
		"conferenceID", targetConference.ID,
		"count", len(talks),
	)

	// Ensure indexes exist
	if err := s.ensureIndexExists(ctx, s.privateIndex); err != nil {
		return fmt.Errorf("failed to ensure private index exists: %w", err)
	}
	if err := s.ensureIndexExists(ctx, s.publicIndex); err != nil {
		return fmt.Errorf("failed to ensure public index exists: %w", err)
	}

	// Index all talks to private index (with privateData merged into data)
	privateTalks := prepareTalksForPrivateIndex(talks)
	if err := s.searchIndex.BulkIndex(ctx, s.privateIndex, privateTalks); err != nil {
		return fmt.Errorf("failed to index to private index: %w", err)
	}

	// Filter approved talks for public index (with private data removed)
	publicTalks := filterApprovedTalksForPublic(talks)

	// Index approved talks to public index
	if err := s.searchIndex.BulkIndex(ctx, s.publicIndex, publicTalks); err != nil {
		return fmt.Errorf("failed to index to public index: %w", err)
	}

	s.logger.Info("conference reindex completed successfully",
		"slug", slug,
		"privateCount", len(talks),
		"publicCount", len(publicTalks),
	)

	return nil
}

// ReindexTalk reindexes a specific talk by its ID.
// It fetches the talk directly and updates both indexes.
func (s *IndexerService) ReindexTalk(ctx context.Context, talkID string) error {
	s.logger.Info("starting reindex for talk", "talkID", talkID)

	// Fetch the talk directly by ID
	targetTalk, err := s.source.GetTalk(ctx, talkID)
	if err != nil {
		return fmt.Errorf("failed to fetch talk %s: %w", talkID, err)
	}

	s.logger.Info("fetched talk",
		"talkID", talkID,
		"conferenceSlug", targetTalk.ConferenceSlug,
	)

	// Ensure indexes exist
	if err := s.ensureIndexExists(ctx, s.privateIndex); err != nil {
		return fmt.Errorf("failed to ensure private index exists: %w", err)
	}
	if err := s.ensureIndexExists(ctx, s.publicIndex); err != nil {
		return fmt.Errorf("failed to ensure public index exists: %w", err)
	}

	// Index to private index (with privateData merged into data)
	privateTalk := targetTalk.ToPrivate()
	if err := s.searchIndex.BulkIndex(ctx, s.privateIndex, []domain.Talk{privateTalk}); err != nil {
		return fmt.Errorf("failed to index to private index: %w", err)
	}

	// Index to public index only if the talk status is public
	if domain.TalkStatus(targetTalk.Status).IsPublic() {
		publicTalk := targetTalk.ToPublic()
		if err := s.searchIndex.BulkIndex(ctx, s.publicIndex, []domain.Talk{publicTalk}); err != nil {
			return fmt.Errorf("failed to index to public index: %w", err)
		}
		s.logger.Info("talk reindex completed successfully",
			"talkID", talkID,
			"indexedToPublic", true,
		)
	} else {
		s.logger.Info("talk reindex completed successfully",
			"talkID", talkID,
			"indexedToPublic", false,
			"status", targetTalk.Status,
		)
	}

	return nil
}

// recreateIndex deletes and recreates an index with the appropriate mapping
func (s *IndexerService) recreateIndex(ctx context.Context, indexName string) error {
	// Delete the index if it exists
	if err := s.searchIndex.DeleteIndex(ctx, indexName); err != nil {
		return fmt.Errorf("failed to delete index %s: %w", indexName, err)
	}

	// Create the index with the appropriate mapping
	mapping := s.getMappingForIndex(indexName)
	if err := s.searchIndex.CreateIndex(ctx, indexName, mapping); err != nil {
		return fmt.Errorf("failed to create index %s: %w", indexName, err)
	}

	return nil
}

// ensureIndexExists creates the index if it doesn't exist
func (s *IndexerService) ensureIndexExists(ctx context.Context, indexName string) error {
	exists, err := s.searchIndex.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if !exists {
		mapping := s.getMappingForIndex(indexName)
		if err := s.searchIndex.CreateIndex(ctx, indexName, mapping); err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
	}

	return nil
}

// getMappingForIndex returns the appropriate mapping for the given index name
func (s *IndexerService) getMappingForIndex(indexName string) string {
	if indexName == s.privateIndex {
		return s.privateIndexMapping
	}
	return s.publicIndexMapping
}

// prepareTalksForPrivateIndex returns talks with privateData merged into data
func prepareTalksForPrivateIndex(talks []domain.Talk) []domain.Talk {
	result := make([]domain.Talk, len(talks))
	for i, talk := range talks {
		result[i] = talk.ToPrivate()
	}
	return result
}

// filterApprovedTalksForPublic returns only approved talks with private data removed
func filterApprovedTalksForPublic(talks []domain.Talk) []domain.Talk {
	approved := make([]domain.Talk, 0)
	for _, talk := range talks {
		if domain.TalkStatus(talk.Status).IsPublic() {
			approved = append(approved, talk.ToPublic())
		}
	}
	return approved
}
