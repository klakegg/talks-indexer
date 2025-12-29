package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestNewHandler(t *testing.T) {
	indexer := &mockIndexer{}
	handler := NewHandler(indexer)

	assert.NotNil(t, handler)
	assert.Equal(t, indexer, handler.indexer)
}

func TestNewHandler_WithNilIndexer(t *testing.T) {
	handler := NewHandler(nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.indexer)
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
