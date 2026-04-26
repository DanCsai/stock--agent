package store

import (
	"context"
	"errors"
	"time"

	"stock_agent/internal/model"
)

var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrAccountExists       = errors.New("account already exists")
	ErrAuthSessionNotFound = errors.New("auth session not found")
)

type Repository interface {
	EnsureSchema(context.Context) error
	CreateUser(context.Context, string, string, string, string) (model.User, error)
	GetUserByAccount(context.Context, string) (model.User, error)
	GetUserByID(context.Context, int64) (model.User, error)
	UpdateUserAvatar(context.Context, int64, string) (model.User, error)
	CreateAuthSession(context.Context, string, int64, time.Time) error
	GetUserByToken(context.Context, string) (model.User, error)
	DeleteAuthSession(context.Context, string) error
	CreateSession(context.Context, int64) (model.Session, error)
	ListSessions(context.Context, int64) ([]model.Session, error)
	GetSession(context.Context, int64, string) (model.Session, error)
	GetMessages(context.Context, int64, string) ([]model.Message, error)
	AddMessage(context.Context, int64, string, string, string) (model.Message, model.Session, error)
}
