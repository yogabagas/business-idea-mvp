package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type session struct {
	token    *oauth2.Token
	created  time.Time
}

var (
	mu       sync.RWMutex
	sessions = make(map[string]*session)
	ttl      = 24 * time.Hour
)

func NewSession(token *oauth2.Token) string {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	mu.Lock()
	sessions[id] = &session{token: token, created: time.Now()}
	mu.Unlock()
	return id
}

func GetToken(sessionID string) *oauth2.Token {
	mu.RLock()
	s := sessions[sessionID]
	mu.RUnlock()
	if s == nil {
		return nil
	}
	if time.Since(s.created) > ttl {
		mu.Lock()
		delete(sessions, sessionID)
		mu.Unlock()
		return nil
	}
	return s.token
}

func DeleteSession(sessionID string) {
	mu.Lock()
	delete(sessions, sessionID)
	mu.Unlock()
}
