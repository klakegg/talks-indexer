package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Session represents an authenticated user session
type Session struct {
	ID        string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store defines the interface for session storage
type Store interface {
	Create(ctx context.Context, email string, ttl time.Duration) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	Delete(ctx context.Context, sessionID string) error
}

// InMemoryStore implements Store with in-memory storage
type InMemoryStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewInMemoryStore creates a new in-memory session store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session for the given email
func (s *InMemoryStore) Create(ctx context.Context, email string, ttl time.Duration) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        id,
		Email:     email,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

// Get retrieves a session by ID, returns nil if expired or not found
func (s *InMemoryStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, nil
	}

	if time.Now().After(session.ExpiresAt) {
		s.Delete(ctx, sessionID)
		return nil, nil
	}

	return session, nil
}

// Delete removes a session
func (s *InMemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return nil
}

// generateSessionID generates a cryptographically secure random session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
