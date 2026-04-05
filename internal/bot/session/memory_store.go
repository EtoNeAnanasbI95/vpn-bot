package session

import (
	"sync"
	"time"
)

const sessionTTL = 15 * time.Minute

type memoryStore struct {
	mu   sync.RWMutex
	data map[int64]*Session
}

func NewMemoryStore() Store {
	return &memoryStore{data: make(map[int64]*Session)}
}

func (s *memoryStore) Get(userID int64) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.data[userID]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Since(sess.UpdatedAt) > sessionTTL {
		s.Clear(userID)
		return nil, false
	}
	return sess, true
}

func (s *memoryStore) Set(userID int64, sess *Session) {
	sess.UpdatedAt = time.Now()
	s.mu.Lock()
	s.data[userID] = sess
	s.mu.Unlock()
}

func (s *memoryStore) Clear(userID int64) {
	s.mu.Lock()
	delete(s.data, userID)
	s.mu.Unlock()
}
