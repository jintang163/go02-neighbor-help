package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"go02-neighbor-help/internal/model"
)

type Session struct {
	Token     string
	UserID    string
	Username  string
	Role      model.UserRole
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (s Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

var ErrInvalidToken = errors.New("invalid or expired token")

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]Session
	now      func() time.Time
	tokenTTL time.Duration
}

func NewSessionManager(tokenTTL time.Duration) *SessionManager {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &SessionManager{
		sessions: make(map[string]Session),
		now:      time.Now,
		tokenTTL: tokenTTL,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return model.TokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func (sm *SessionManager) Create(u model.User) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	now := sm.now()
	sess := Session{
		Token:     token,
		UserID:    u.ID,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.tokenTTL),
	}
	sm.mu.Lock()
	sm.sessions[token] = sess
	sm.mu.Unlock()
	return token, nil
}

func (sm *SessionManager) Get(token string) (Session, error) {
	if token == "" {
		return Session{}, ErrInvalidToken
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, ok := sm.sessions[token]
	if !ok {
		return Session{}, ErrInvalidToken
	}
	if sess.Expired() {
		delete(sm.sessions, token)
		return Session{}, ErrInvalidToken
	}
	return sess, nil
}

func (sm *SessionManager) Invalidate(token string) {
	if token == "" {
		return
	}
	sm.mu.Lock()
	delete(sm.sessions, token)
	sm.mu.Unlock()
}

func (sm *SessionManager) CleanupExpired() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	count := 0
	for k, s := range sm.sessions {
		if s.Expired() {
			delete(sm.sessions, k)
			count++
		}
	}
	return count
}

func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

func (sm *SessionManager) InvalidateByUser(userID string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	count := 0
	for k, s := range sm.sessions {
		if s.UserID == userID {
			delete(sm.sessions, k)
			count++
		}
	}
	return count
}
