package api

import (
	"context"
	"errors"
	"testing"

	"github.com/javaBin/talks-indexer/internal/config"
	"github.com/stretchr/testify/assert"
)

// testContext returns a context with test configuration
func testContext() context.Context {
	cfg := &config.Config{
		ApplicationConfig: config.ApplicationConfig{
			Mode: config.ModeDevelopment,
		},
	}
	return config.WithConfig(context.Background(), cfg)
}

// mockIndexer is a mock implementation of the Indexer interface for testing
type mockIndexer struct {
	reindexAllFunc        func(ctx context.Context) error
	reindexConferenceFunc func(ctx context.Context, slug string) error
	reindexTalkFunc       func(ctx context.Context, talkID string) error
}

func (m *mockIndexer) ReindexAll(ctx context.Context) error {
	if m.reindexAllFunc != nil {
		return m.reindexAllFunc(ctx)
	}
	return nil
}

func (m *mockIndexer) ReindexConference(ctx context.Context, slug string) error {
	if m.reindexConferenceFunc != nil {
		return m.reindexConferenceFunc(ctx, slug)
	}
	return nil
}

func (m *mockIndexer) ReindexTalk(ctx context.Context, talkID string) error {
	if m.reindexTalkFunc != nil {
		return m.reindexTalkFunc(ctx, talkID)
	}
	return nil
}

func TestNew(t *testing.T) {
	ctx := testContext()
	indexer := &mockIndexer{}
	adapter := New(ctx, indexer)

	assert.NotNil(t, adapter)
	assert.Equal(t, indexer, adapter.indexer)
}

func TestNew_WithNilIndexer(t *testing.T) {
	ctx := testContext()
	adapter := New(ctx, nil)

	assert.NotNil(t, adapter)
	assert.Nil(t, adapter.indexer)
}

func TestMockIndexer_ReindexAll_Default(t *testing.T) {
	indexer := &mockIndexer{}
	err := indexer.ReindexAll(context.Background())

	assert.NoError(t, err)
}

func TestMockIndexer_ReindexAll_WithError(t *testing.T) {
	expectedError := errors.New("reindex error")
	indexer := &mockIndexer{
		reindexAllFunc: func(ctx context.Context) error {
			return expectedError
		},
	}

	err := indexer.ReindexAll(context.Background())
	assert.Equal(t, expectedError, err)
}

func TestMockIndexer_ReindexConference_Default(t *testing.T) {
	indexer := &mockIndexer{}
	err := indexer.ReindexConference(context.Background(), "test-slug")

	assert.NoError(t, err)
}

func TestMockIndexer_ReindexConference_WithError(t *testing.T) {
	expectedError := errors.New("conference reindex error")
	indexer := &mockIndexer{
		reindexConferenceFunc: func(ctx context.Context, slug string) error {
			return expectedError
		},
	}

	err := indexer.ReindexConference(context.Background(), "test-slug")
	assert.Equal(t, expectedError, err)
}
