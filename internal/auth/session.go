package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type session struct {
	expiresAt time.Time
}

// SessionManager issues opaque tokens with a sliding expiry. This is what
// makes logout and idle-timeout possible — HTTP Basic Auth can't do either,
// since browsers cache Basic credentials until the browser itself closes.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]session
	timeout  time.Duration
}

func NewSessionManager(timeout time.Duration) *SessionManager {
	sm := &SessionManager{sessions: map[string]session{}, timeout: timeout}
	go sm.reap()
	return sm
}

func (sm *SessionManager) Create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	sm.mu.Lock()
	sm.sessions[token] = session{expiresAt: time.Now().Add(sm.timeout)}
	sm.mu.Unlock()

	return token, nil
}

// Validate checks the token and, on success, slides the expiry forward —
// active use keeps a session alive; idle time lets it expire on its own.
func (sm *SessionManager) Validate(token string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[token]
	if !ok || time.Now().After(s.expiresAt) {
		delete(sm.sessions, token)
		return false
	}
	s.expiresAt = time.Now().Add(sm.timeout)
	sm.sessions[token] = s
	return true
}

func (sm *SessionManager) Revoke(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

func (sm *SessionManager) reap() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for token, s := range sm.sessions {
			if now.After(s.expiresAt) {
				delete(sm.sessions, token)
			}
		}
		sm.mu.Unlock()
	}
}
