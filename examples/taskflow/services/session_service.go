package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	flin "github.com/skshohagmiah/flin/clients/go"
)

// SessionService handles session management using KV store
type SessionService struct {
	client *flin.Client
}

// NewSessionService creates a new session service
func NewSessionService(client *flin.Client) *SessionService {
	return &SessionService{client: client}
}

// Session represents a user session
type Session struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

// CreateSession creates a new session for a user
func (s *SessionService) CreateSession(userID string) (*Session, error) {
	session := &Session{
		SessionID: uuid.New().String(),
		UserID:    userID,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	// Store session in KV store
	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("session:%s", session.SessionID)
	err = s.client.KV.Set(key, data)
	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := s.client.KV.Get(key)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if time.Now().Unix() > session.ExpiresAt {
		s.DeleteSession(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.KV.Delete(key)
}
