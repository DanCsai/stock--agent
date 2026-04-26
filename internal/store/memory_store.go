package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"stock_agent/internal/model"
)

type MemoryStore struct {
	mu            sync.RWMutex
	nextUserID    int64
	users         map[int64]model.User
	accountToUser map[string]int64
	authTokens    map[string]authToken
	sessions      map[string]model.Session
	messages      map[string][]model.Message
}

type authToken struct {
	UserID    int64
	ExpiresAt time.Time
}

func NewMemory() *MemoryStore {
	return &MemoryStore{
		nextUserID:    1,
		users:         map[int64]model.User{},
		accountToUser: map[string]int64{},
		authTokens:    map[string]authToken{},
		sessions:      map[string]model.Session{},
		messages:      map[string][]model.Message{},
	}
}

func (s *MemoryStore) EnsureSchema(context.Context) error { return nil }

func (s *MemoryStore) CreateUser(_ context.Context, account, username, passwordHash, avatar string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account = strings.TrimSpace(account)
	if _, ok := s.accountToUser[account]; ok {
		return model.User{}, ErrAccountExists
	}

	now := time.Now().UTC()
	user := model.User{
		ID:           s.nextUserID,
		Account:      account,
		Username:     strings.TrimSpace(username),
		PasswordHash: passwordHash,
		Avatar:       strings.TrimSpace(avatar),
		CreatedAt:    nowString(now),
		UpdatedAt:    nowString(now),
	}
	s.nextUserID++
	s.users[user.ID] = user
	s.accountToUser[user.Account] = user.ID
	return user, nil
}

func (s *MemoryStore) GetUserByAccount(_ context.Context, account string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.accountToUser[strings.TrimSpace(account)]
	if !ok {
		return model.User{}, ErrUserNotFound
	}
	return s.users[id], nil
}

func (s *MemoryStore) GetUserByID(_ context.Context, userID int64) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userID]
	if !ok {
		return model.User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) UpdateUserAvatar(_ context.Context, userID int64, avatar string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return model.User{}, ErrUserNotFound
	}
	user.Avatar = strings.TrimSpace(avatar)
	user.UpdatedAt = nowString(time.Now().UTC())
	s.users[userID] = user
	return user, nil
}

func (s *MemoryStore) CreateAuthSession(_ context.Context, token string, userID int64, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authTokens[token] = authToken{UserID: userID, ExpiresAt: expiresAt.UTC()}
	return nil
}

func (s *MemoryStore) GetUserByToken(_ context.Context, token string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.authTokens[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return model.User{}, ErrAuthSessionNotFound
	}
	user, ok := s.users[session.UserID]
	if !ok {
		return model.User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) DeleteAuthSession(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.authTokens[token]; !ok {
		return ErrAuthSessionNotFound
	}
	delete(s.authTokens, token)
	return nil
}

func (s *MemoryStore) CreateSession(_ context.Context, userID int64) (model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, err := randomID("sess")
	if err != nil {
		return model.Session{}, err
	}
	now := time.Now().UTC()
	session := model.Session{
		ID:        sessionID,
		UserID:    userID,
		Title:     "新会话",
		CreatedAt: nowString(now),
		UpdatedAt: nowString(now),
	}
	s.sessions[sessionID] = session
	s.messages[sessionID] = []model.Message{}
	return session, nil
}

func (s *MemoryStore) ListSessions(_ context.Context, userID int64) ([]model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []model.Session{}
	for _, session := range s.sessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt > result[j].UpdatedAt })
	return result, nil
}

func (s *MemoryStore) GetSession(_ context.Context, userID int64, sessionID string) (model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.UserID != userID {
		return model.Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *MemoryStore) GetMessages(_ context.Context, userID int64, sessionID string) ([]model.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.UserID != userID {
		return nil, ErrSessionNotFound
	}
	return append([]model.Message(nil), s.messages[sessionID]...), nil
}

func (s *MemoryStore) AddMessage(_ context.Context, userID int64, sessionID, role, content string) (model.Message, model.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.UserID != userID {
		return model.Message{}, model.Session{}, ErrSessionNotFound
	}
	messageID, err := randomID("msg")
	if err != nil {
		return model.Message{}, model.Session{}, err
	}
	now := time.Now().UTC()
	message := model.Message{
		ID:        messageID,
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		Content:   strings.TrimSpace(content),
		CreatedAt: nowString(now),
	}
	s.messages[sessionID] = append(s.messages[sessionID], message)
	if session.Title == "新会话" && role == "user" {
		session.Title = buildTitle(message.Content)
	}
	session.LastMessagePreview = buildPreview(message.Content)
	session.UpdatedAt = nowString(now)
	s.sessions[sessionID] = session
	return message, session, nil
}
