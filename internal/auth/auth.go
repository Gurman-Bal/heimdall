package auth

import (
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"heimdall/internal/storage"
)

const (
	settingUsername     = "auth_username"
	settingPasswordHash = "auth_password_hash"
)

type Store struct {
	mu           sync.RWMutex
	username     string
	passwordHash []byte
	db           *storage.Store
}

func Load(db *storage.Store, defaultUser, defaultPass string) (*Store, error) {
	s := &Store{db: db}

	username, found, err := db.GetSetting(settingUsername)
	if err != nil {
		return nil, err
	}
	if !found {
		username = defaultUser
		if err := db.SetSetting(settingUsername, username); err != nil {
			return nil, err
		}
	}
	s.username = username

	hashStr, found, err := db.GetSetting(settingPasswordHash)
	if err != nil {
		return nil, err
	}
	if !found {
		if defaultPass == "" {
			return nil, fmt.Errorf("no password configured — set HEIMDALL_AUTH_PASS before first run")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		if err := db.SetSetting(settingPasswordHash, string(hash)); err != nil {
			return nil, err
		}
		s.passwordHash = hash
		slog.Info("auth credentials seeded from environment on first run")
	} else {
		s.passwordHash = []byte(hashStr)
	}

	return s, nil
}

func (s *Store) Verify(username, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if username != s.username {
		return false
	}
	return bcrypt.CompareHashAndPassword(s.passwordHash, []byte(password)) == nil
}

func (s *Store) ChangePassword(currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if bcrypt.CompareHashAndPassword(s.passwordHash, []byte(currentPassword)) != nil {
		return fmt.Errorf("current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.db.SetSetting(settingPasswordHash, string(hash)); err != nil {
		return err
	}
	s.passwordHash = hash
	slog.Warn("password changed via api")
	return nil
}
