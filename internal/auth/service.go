package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const SessionCookieName = "fund_session"

type Service struct {
	defaultAvatar string
}

func NewService(defaultAvatar string) *Service {
	return &Service{defaultAvatar: defaultAvatar}
}

func (s *Service) DefaultAvatar() string {
	return s.defaultAvatar
}

func (s *Service) HashPassword(password string) (string, error) {
	salt, err := randomHex(16)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(salt + ":" + password))
	return salt + ":" + hex.EncodeToString(hash[:]), nil
}

func (s *Service) VerifyPassword(storedHash, password string) bool {
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return false
	}
	hash := sha256.Sum256([]byte(parts[0] + ":" + password))
	return hex.EncodeToString(hash[:]) == parts[1]
}

func (s *Service) NewSessionToken() (string, error) {
	return randomHex(32)
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("failed to generate secure random bytes")
	}
	return hex.EncodeToString(buf), nil
}
