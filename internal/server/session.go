package server

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Session struct {
	Token      string
	OperatorID string
	Role       string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	ss := &SessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
	go ss.cleanup()
	return ss
}

func (ss *SessionStore) Create(operatorID, role string) (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := "rt_sess_" + base64.RawURLEncoding.EncodeToString(b)

	now := time.Now()
	s := &Session{
		Token:      token,
		OperatorID: operatorID,
		Role:       role,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ss.ttl),
		LastSeenAt: now,
	}

	ss.mu.Lock()
	ss.sessions[token] = s
	ss.mu.Unlock()

	return s, nil
}

func (ss *SessionStore) Get(token string) *Session {
	ss.mu.RLock()
	s, ok := ss.sessions[token]
	ss.mu.RUnlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		if ok {
			ss.mu.Lock()
			delete(ss.sessions, token)
			ss.mu.Unlock()
		}
		return nil
	}

	ss.mu.Lock()
	s.LastSeenAt = time.Now()
	ss.mu.Unlock()

	return s
}

func (ss *SessionStore) Delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

// DeleteByOperator removes all sessions for an operator (e.g. on key rotation).
func (ss *SessionStore) DeleteByOperator(operatorID string) {
	ss.mu.Lock()
	for token, s := range ss.sessions {
		if s.OperatorID == operatorID {
			delete(ss.sessions, token)
		}
	}
	ss.mu.Unlock()
}

func (ss *SessionStore) ExpireAll() {
	ss.mu.Lock()
	ss.sessions = make(map[string]*Session)
	ss.mu.Unlock()
}

func (ss *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now()
		ss.mu.Lock()
		for token, s := range ss.sessions {
			if now.After(s.ExpiresAt) {
				delete(ss.sessions, token)
			}
		}
		ss.mu.Unlock()
	}
}
