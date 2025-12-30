package app

import (
	"context"
	"errors"
	"testing"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/javaBin/talks-indexer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPrivateMapping = `{"mappings":{"private":true}}`
	testPublicMapping  = `{"mappings":{"public":true}}`
)

// testIndexConfig creates a test config with index names
func testIndexConfig() *config.Config {
	return &config.Config{
		Index: config.IndexConfig{
			Private: "private",
			Public:  "public",
		},
	}
}

// mockTalkSource is a mock implementation of ports.TalkSource
type mockTalkSource struct {
	getConferencesFunc func(ctx context.Context) ([]domain.Conference, error)
	getTalksFunc       func(ctx context.Context, conferenceID string) ([]domain.Talk, error)
	getTalkFunc        func(ctx context.Context, talkID string) (*domain.Talk, error)
}

func (m *mockTalkSource) GetConferences(ctx context.Context) ([]domain.Conference, error) {
	if m.getConferencesFunc != nil {
		return m.getConferencesFunc(ctx)
	}
	return nil, nil
}

func (m *mockTalkSource) GetTalks(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
	if m.getTalksFunc != nil {
		return m.getTalksFunc(ctx, conferenceID)
	}
	return nil, nil
}

func (m *mockTalkSource) GetTalk(ctx context.Context, talkID string) (*domain.Talk, error) {
	if m.getTalkFunc != nil {
		return m.getTalkFunc(ctx, talkID)
	}
	return nil, nil
}

// mockSearchIndex is a mock implementation of ports.SearchIndex
type mockSearchIndex struct {
	bulkIndexFunc    func(ctx context.Context, indexName string, talks []domain.Talk) error
	deleteIndexFunc  func(ctx context.Context, indexName string) error
	createIndexFunc  func(ctx context.Context, indexName string, mapping string) error
	indexExistsFunc  func(ctx context.Context, indexName string) (bool, error)
	bulkIndexCalls   []bulkIndexCall
	deleteIndexCalls []string
	createIndexCalls []string
}

type bulkIndexCall struct {
	IndexName string
	Talks     []domain.Talk
}

func (m *mockSearchIndex) BulkIndex(ctx context.Context, indexName string, talks []domain.Talk) error {
	m.bulkIndexCalls = append(m.bulkIndexCalls, bulkIndexCall{IndexName: indexName, Talks: talks})
	if m.bulkIndexFunc != nil {
		return m.bulkIndexFunc(ctx, indexName, talks)
	}
	return nil
}

func (m *mockSearchIndex) DeleteIndex(ctx context.Context, indexName string) error {
	m.deleteIndexCalls = append(m.deleteIndexCalls, indexName)
	if m.deleteIndexFunc != nil {
		return m.deleteIndexFunc(ctx, indexName)
	}
	return nil
}

func (m *mockSearchIndex) CreateIndex(ctx context.Context, indexName string, mapping string) error {
	m.createIndexCalls = append(m.createIndexCalls, indexName)
	if m.createIndexFunc != nil {
		return m.createIndexFunc(ctx, indexName, mapping)
	}
	return nil
}

func (m *mockSearchIndex) IndexExists(ctx context.Context, indexName string) (bool, error) {
	if m.indexExistsFunc != nil {
		return m.indexExistsFunc(ctx, indexName)
	}
	return true, nil
}

func TestNewIndexerService(t *testing.T) {
	t.Run("with context config", func(t *testing.T) {
		source := &mockTalkSource{}
		index := &mockSearchIndex{}

		cfg := testIndexConfig()
		ctx := config.WithConfig(context.Background(), cfg)

		service := NewIndexerService(ctx, source, index, testPrivateMapping, testPublicMapping)

		assert.NotNil(t, service)
		assert.Equal(t, source, service.source)
		assert.Equal(t, index, service.searchIndex)
		assert.Equal(t, "private", service.privateIndex)
		assert.Equal(t, "public", service.publicIndex)
		assert.Equal(t, testPrivateMapping, service.privateIndexMapping)
		assert.Equal(t, testPublicMapping, service.publicIndexMapping)
	})

	t.Run("panics when config not in context", func(t *testing.T) {
		source := &mockTalkSource{}
		index := &mockSearchIndex{}
		ctx := context.Background()

		assert.Panics(t, func() {
			NewIndexerService(ctx, source, index, testPrivateMapping, testPublicMapping)
		})
	})
}

func TestNewIndexerServiceWithConfig(t *testing.T) {
	source := &mockTalkSource{}
	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)

	assert.NotNil(t, service)
	assert.Equal(t, source, service.source)
	assert.Equal(t, index, service.searchIndex)
	assert.Equal(t, "private", service.privateIndex)
	assert.Equal(t, "public", service.publicIndex)
	assert.Equal(t, testPrivateMapping, service.privateIndexMapping)
	assert.Equal(t, testPublicMapping, service.publicIndexMapping)
}

func TestReindexAll_Success(t *testing.T) {
	conferences := []domain.Conference{
		{ID: "conf-1", Name: "JavaZone 2024", Slug: "javazone2024"},
	}

	talks := []domain.Talk{
		{ID: "talk-1", ConferenceID: "conf-1", Status: "APPROVED", Data: map[string]interface{}{"title": "Talk 1"}},
		{ID: "talk-2", ConferenceID: "conf-1", Status: "SUBMITTED", Data: map[string]interface{}{"title": "Talk 2"}},
		{ID: "talk-3", ConferenceID: "conf-1", Status: "APPROVED", Data: map[string]interface{}{"title": "Talk 3"}},
	}

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return conferences, nil
		},
		getTalksFunc: func(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
			return talks, nil
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexAll(context.Background())

	require.NoError(t, err)

	// Verify indexes were recreated
	assert.Contains(t, index.deleteIndexCalls, "private")
	assert.Contains(t, index.deleteIndexCalls, "public")
	assert.Contains(t, index.createIndexCalls, "private")
	assert.Contains(t, index.createIndexCalls, "public")

	// Verify bulk index calls
	require.Len(t, index.bulkIndexCalls, 2)

	// First call should be private index with all talks
	privateCall := index.bulkIndexCalls[0]
	assert.Equal(t, "private", privateCall.IndexName)
	assert.Len(t, privateCall.Talks, 3)

	// Second call should be public index with only approved talks
	publicCall := index.bulkIndexCalls[1]
	assert.Equal(t, "public", publicCall.IndexName)
	assert.Len(t, publicCall.Talks, 2)
}

func TestReindexAll_NoConferences(t *testing.T) {
	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return []domain.Conference{}, nil
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexAll(context.Background())

	require.NoError(t, err)

	// Verify indexes were recreated but no bulk indexing happened
	assert.Contains(t, index.deleteIndexCalls, "private")
	assert.Contains(t, index.deleteIndexCalls, "public")
	assert.Empty(t, index.bulkIndexCalls)
}

func TestReindexAll_FetchConferencesError(t *testing.T) {
	expectedErr := errors.New("connection error")

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return nil, expectedErr
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexAll(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch conferences")
}

func TestReindexAll_FetchTalksError_ContinuesWithOtherConferences(t *testing.T) {
	conferences := []domain.Conference{
		{ID: "conf-1", Name: "Conference 1", Slug: "conf1"},
		{ID: "conf-2", Name: "Conference 2", Slug: "conf2"},
	}

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return conferences, nil
		},
		getTalksFunc: func(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
			if conferenceID == "conf-1" {
				return nil, errors.New("error fetching talks")
			}
			return []domain.Talk{
				{ID: "talk-1", Status: "APPROVED"},
			}, nil
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexAll(context.Background())

	// Should not return error, just log and continue
	require.NoError(t, err)

	// Should have indexed talks from conf-2
	require.Len(t, index.bulkIndexCalls, 2)
}

func TestReindexConference_Success(t *testing.T) {
	conferences := []domain.Conference{
		{ID: "conf-1", Name: "JavaZone 2024", Slug: "javazone2024"},
	}

	talks := []domain.Talk{
		{ID: "talk-1", ConferenceID: "conf-1", Status: "APPROVED", Data: map[string]interface{}{"title": "Talk 1"}},
		{ID: "talk-2", ConferenceID: "conf-1", Status: "SUBMITTED", Data: map[string]interface{}{"title": "Talk 2"}},
	}

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return conferences, nil
		},
		getTalksFunc: func(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
			return talks, nil
		},
	}

	index := &mockSearchIndex{
		indexExistsFunc: func(ctx context.Context, indexName string) (bool, error) {
			return true, nil
		},
	}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexConference(context.Background(), "javazone2024")

	require.NoError(t, err)

	// Should not recreate indexes, just ensure they exist
	assert.Empty(t, index.deleteIndexCalls)

	// Verify bulk index calls
	require.Len(t, index.bulkIndexCalls, 2)

	privateCall := index.bulkIndexCalls[0]
	assert.Equal(t, "private", privateCall.IndexName)
	assert.Len(t, privateCall.Talks, 2)

	publicCall := index.bulkIndexCalls[1]
	assert.Equal(t, "public", publicCall.IndexName)
	assert.Len(t, publicCall.Talks, 1) // Only approved
}

func TestReindexConference_NotFound(t *testing.T) {
	conferences := []domain.Conference{
		{ID: "conf-1", Name: "JavaZone 2024", Slug: "javazone2024"},
	}

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return conferences, nil
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexConference(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conference not found with slug")
}

func TestReindexConference_CreateIndexIfNotExists(t *testing.T) {
	conferences := []domain.Conference{
		{ID: "conf-1", Name: "Test", Slug: "test"},
	}

	source := &mockTalkSource{
		getConferencesFunc: func(ctx context.Context) ([]domain.Conference, error) {
			return conferences, nil
		},
		getTalksFunc: func(ctx context.Context, conferenceID string) ([]domain.Talk, error) {
			return []domain.Talk{}, nil
		},
	}

	index := &mockSearchIndex{
		indexExistsFunc: func(ctx context.Context, indexName string) (bool, error) {
			return false, nil
		},
	}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexConference(context.Background(), "test")

	require.NoError(t, err)

	// Should have created both indexes
	assert.Contains(t, index.createIndexCalls, "private")
	assert.Contains(t, index.createIndexCalls, "public")
}

func TestReindexTalk_ApprovedTalk(t *testing.T) {
	talk := &domain.Talk{
		ID:             "talk-1",
		ConferenceID:   "conf-1",
		ConferenceSlug: "javazone2024",
		Status:         "APPROVED",
		Data:           map[string]interface{}{"title": "Test Talk"},
	}

	source := &mockTalkSource{
		getTalkFunc: func(ctx context.Context, talkID string) (*domain.Talk, error) {
			return talk, nil
		},
	}

	index := &mockSearchIndex{
		indexExistsFunc: func(ctx context.Context, indexName string) (bool, error) {
			return true, nil
		},
	}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexTalk(context.Background(), "talk-1")

	require.NoError(t, err)

	// Should have indexed to both private and public (since approved)
	require.Len(t, index.bulkIndexCalls, 2)
	assert.Equal(t, "private", index.bulkIndexCalls[0].IndexName)
	assert.Equal(t, "public", index.bulkIndexCalls[1].IndexName)
}

func TestReindexTalk_NonApprovedTalk(t *testing.T) {
	talk := &domain.Talk{
		ID:             "talk-1",
		ConferenceID:   "conf-1",
		ConferenceSlug: "javazone2024",
		Status:         "SUBMITTED",
		Data:           map[string]interface{}{"title": "Test Talk"},
	}

	source := &mockTalkSource{
		getTalkFunc: func(ctx context.Context, talkID string) (*domain.Talk, error) {
			return talk, nil
		},
	}

	index := &mockSearchIndex{
		indexExistsFunc: func(ctx context.Context, indexName string) (bool, error) {
			return true, nil
		},
	}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexTalk(context.Background(), "talk-1")

	require.NoError(t, err)

	// Should have indexed only to private (since not approved)
	require.Len(t, index.bulkIndexCalls, 1)
	assert.Equal(t, "private", index.bulkIndexCalls[0].IndexName)
}

func TestReindexTalk_TalkNotFound(t *testing.T) {
	source := &mockTalkSource{
		getTalkFunc: func(ctx context.Context, talkID string) (*domain.Talk, error) {
			return nil, errors.New("talk not found")
		},
	}

	index := &mockSearchIndex{}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexTalk(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch talk")
}

func TestReindexTalk_CreateIndexIfNotExists(t *testing.T) {
	talk := &domain.Talk{
		ID:             "talk-1",
		ConferenceID:   "conf-1",
		ConferenceSlug: "javazone2024",
		Status:         "SUBMITTED",
		Data:           map[string]interface{}{"title": "Test Talk"},
	}

	source := &mockTalkSource{
		getTalkFunc: func(ctx context.Context, talkID string) (*domain.Talk, error) {
			return talk, nil
		},
	}

	index := &mockSearchIndex{
		indexExistsFunc: func(ctx context.Context, indexName string) (bool, error) {
			return false, nil
		},
	}

	service := NewIndexerServiceWithConfig(source, index, "private", "public", testPrivateMapping, testPublicMapping)
	err := service.ReindexTalk(context.Background(), "talk-1")

	require.NoError(t, err)

	// Should have created both indexes
	assert.Contains(t, index.createIndexCalls, "private")
	assert.Contains(t, index.createIndexCalls, "public")
}

func TestFilterApprovedTalksForPublic(t *testing.T) {
	talks := []domain.Talk{
		{ID: "1", Status: "APPROVED"},
		{ID: "2", Status: "SUBMITTED"},
		{ID: "3", Status: "APPROVED"},
		{ID: "4", Status: "REJECTED"},
		{ID: "5", Status: "DRAFT"},
	}

	approved := filterApprovedTalksForPublic(talks)

	assert.Len(t, approved, 2)
	assert.Equal(t, "1", approved[0].ID)
	assert.Equal(t, "3", approved[1].ID)
}

func TestFilterApprovedTalksForPublic_Empty(t *testing.T) {
	talks := []domain.Talk{}
	approved := filterApprovedTalksForPublic(talks)

	assert.NotNil(t, approved)
	assert.Len(t, approved, 0)
}

func TestFilterApprovedTalksForPublic_NoApproved(t *testing.T) {
	talks := []domain.Talk{
		{ID: "1", Status: "SUBMITTED"},
		{ID: "2", Status: "REJECTED"},
	}

	approved := filterApprovedTalksForPublic(talks)

	assert.NotNil(t, approved)
	assert.Len(t, approved, 0)
}
