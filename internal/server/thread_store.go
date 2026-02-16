package server

import (
	"sync"
	"time"
)

// ThreadStore holds session_id to thread_ts mappings in memory.
type ThreadStore struct {
	mu      sync.RWMutex
	threads map[string]threadEntry
}

type threadEntry struct {
	ThreadTS  string
	CreatedAt time.Time
}

// NewThreadStore creates a new empty ThreadStore.
func NewThreadStore() *ThreadStore {
	return &ThreadStore{
		threads: make(map[string]threadEntry),
	}
}

// Get returns the thread_ts for a session, or empty string if not found.
func (s *ThreadStore) Get(sessionID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.threads[sessionID].ThreadTS
}

// Set stores the thread_ts for a session.
func (s *ThreadStore) Set(sessionID, threadTS string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.threads[sessionID] = threadEntry{
		ThreadTS:  threadTS,
		CreatedAt: time.Now(),
	}
}

// CleanOlderThan removes entries older than maxAge.
func (s *ThreadStore) CleanOlderThan(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for id, entry := range s.threads {
		if entry.CreatedAt.Before(cutoff) {
			delete(s.threads, id)
		}
	}
}
