// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package service provides business logic for the Image Sync application.
package service

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// SessionInfo stores information about a user session.
type SessionInfo struct {
	UserID   string
	Groups   []string
	Email    string
	ExpireAt time.Time
}

// SessionService manages user sessions.
type SessionService struct {
	sessions map[string]*SessionInfo
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSessionService creates a new session service.
func NewSessionService(ttl time.Duration) *SessionService {
	s := &SessionService{
		sessions: make(map[string]*SessionInfo),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go s.cleanup()

	return s
}

// CreateSession creates a new session and returns the session ID.
func (s *SessionService) CreateSession(userID, email string, groups []string) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = &SessionInfo{
		UserID:   userID,
		Groups:   groups,
		Email:    email,
		ExpireAt: time.Now().Add(s.ttl),
	}

	return sessionID, nil
}

// GetSession retrieves session information by session ID.
// Returns interface{} to satisfy middleware.SessionValidator interface.
func (s *SessionService) GetSession(sessionID string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpireAt) {
		return nil, false
	}

	return session, true
}

// GetSessionInfo retrieves typed session information by session ID.
func (s *SessionService) GetSessionInfo(sessionID string) (*SessionInfo, bool) {
	val, exists := s.GetSession(sessionID)
	if !exists {
		return nil, false
	}
	return val.(*SessionInfo), true
}

// DeleteSession removes a session.
func (s *SessionService) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
}

// RefreshSession extends the session expiration time.
func (s *SessionService) RefreshSession(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return false
	}

	session.ExpireAt = time.Now().Add(s.ttl)
	return true
}

// cleanup removes expired sessions periodically.
func (s *SessionService) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if now.After(session.ExpireAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// generateSessionID generates a cryptographically secure random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
