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

// SetTimeout changes the idle timeout for future validations. Existing
// sessions keep whatever expiry they already had until their next
// successful Validate call, at which point the new timeout applies.
func (sm *SessionManager) SetTimeout(timeout time.Duration) {
	sm.mu.Lock()
	sm.timeout = timeout
	sm.mu.Unlock()
}

func (sm *SessionManager) Timeout() time.Duration {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.timeout
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
