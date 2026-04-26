package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"stock_agent/internal/model"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) EnsureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			account VARCHAR(128) NOT NULL UNIQUE,
			username VARCHAR(128) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			avatar VARCHAR(64) NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			token VARCHAR(128) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			expires_at DATETIME(6) NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			CONSTRAINT fk_auth_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id VARCHAR(64) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			last_message_preview VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			INDEX idx_chat_sessions_user_updated (user_id, updated_at),
			CONSTRAINT fk_chat_session_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id VARCHAR(64) PRIMARY KEY,
			session_id VARCHAR(64) NOT NULL,
			user_id BIGINT NOT NULL,
			role VARCHAR(32) NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			INDEX idx_chat_messages_session_created (session_id, created_at),
			CONSTRAINT fk_chat_message_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
			CONSTRAINT fk_chat_message_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

func (s *MySQLStore) CreateUser(ctx context.Context, account, username, passwordHash, avatar string) (model.User, error) {
	account = strings.TrimSpace(account)
	username = strings.TrimSpace(username)
	avatar = strings.TrimSpace(avatar)

	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (account, username, password_hash, avatar) VALUES (?, ?, ?, ?)`,
		account,
		username,
		passwordHash,
		avatar,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return model.User{}, ErrAccountExists
		}
		return model.User{}, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return model.User{}, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *MySQLStore) GetUserByAccount(ctx context.Context, account string) (model.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, account, username, password_hash, avatar, created_at, updated_at FROM users WHERE account = ?`, strings.TrimSpace(account))
	return scanUser(row)
}

func (s *MySQLStore) GetUserByID(ctx context.Context, userID int64) (model.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, account, username, password_hash, avatar, created_at, updated_at FROM users WHERE id = ?`, userID)
	return scanUser(row)
}

func (s *MySQLStore) UpdateUserAvatar(ctx context.Context, userID int64, avatar string) (model.User, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET avatar = ? WHERE id = ?`, strings.TrimSpace(avatar), userID); err != nil {
		return model.User{}, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *MySQLStore) CreateAuthSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_sessions (token, user_id, expires_at) VALUES (?, ?, ?)`, token, userID, expiresAt.UTC())
	return err
}

func (s *MySQLStore) GetUserByToken(ctx context.Context, token string) (model.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.account, u.username, u.password_hash, u.avatar, u.created_at, u.updated_at
		FROM auth_sessions a
		JOIN users u ON u.id = a.user_id
		WHERE a.token = ? AND a.expires_at > UTC_TIMESTAMP(6)
	`, token)
	return scanUser(row)
}

func (s *MySQLStore) DeleteAuthSession(ctx context.Context, token string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token = ?`, token)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrAuthSessionNotFound
	}
	return nil
}

func (s *MySQLStore) CreateSession(ctx context.Context, userID int64) (model.Session, error) {
	sessionID, err := randomID("sess")
	if err != nil {
		return model.Session{}, err
	}

	if _, err := s.db.ExecContext(ctx, `INSERT INTO chat_sessions (id, user_id, title) VALUES (?, ?, ?)`, sessionID, userID, "新会话"); err != nil {
		return model.Session{}, err
	}

	return s.GetSession(ctx, userID, sessionID)
}

func (s *MySQLStore) ListSessions(ctx context.Context, userID int64) ([]model.Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, title, last_message_preview, created_at, updated_at FROM chat_sessions WHERE user_id = ? ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []model.Session{}
	for rows.Next() {
		session, err := scanSessionRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *MySQLStore) GetSession(ctx context.Context, userID int64, sessionID string) (model.Session, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, title, last_message_preview, created_at, updated_at FROM chat_sessions WHERE id = ? AND user_id = ?`, sessionID, userID)
	return scanSession(row)
}

func (s *MySQLStore) GetMessages(ctx context.Context, userID int64, sessionID string) ([]model.Message, error) {
	if _, err := s.GetSession(ctx, userID, sessionID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, user_id, role, content, created_at FROM chat_messages WHERE session_id = ? AND user_id = ? ORDER BY created_at ASC`, sessionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []model.Message{}
	for rows.Next() {
		message, err := scanMessageRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *MySQLStore) AddMessage(ctx context.Context, userID int64, sessionID, role, content string) (model.Message, model.Session, error) {
	session, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return model.Message{}, model.Session{}, err
	}

	messageID, err := randomID("msg")
	if err != nil {
		return model.Message{}, model.Session{}, err
	}

	content = strings.TrimSpace(content)
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chat_messages (id, session_id, user_id, role, content) VALUES (?, ?, ?, ?, ?)`,
		messageID,
		sessionID,
		userID,
		role,
		content,
	); err != nil {
		return model.Message{}, model.Session{}, err
	}

	title := session.Title
	if title == "新会话" && role == "user" {
		title = buildTitle(content)
	}

	if _, err := s.db.ExecContext(
		ctx,
		`UPDATE chat_sessions SET title = ?, last_message_preview = ?, updated_at = UTC_TIMESTAMP(6) WHERE id = ? AND user_id = ?`,
		title,
		buildPreview(content),
		sessionID,
		userID,
	); err != nil {
		return model.Message{}, model.Session{}, err
	}

	updatedSession, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return model.Message{}, model.Session{}, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, user_id, role, content, created_at FROM chat_messages WHERE id = ?`, messageID)
	if err != nil {
		return model.Message{}, model.Session{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return model.Message{}, model.Session{}, ErrSessionNotFound
	}
	message, err := scanMessageRows(rows)
	if err != nil {
		return model.Message{}, model.Session{}, err
	}
	return message, updatedSession, nil
}

func buildTitle(content string) string {
	trimmed := []rune(strings.TrimSpace(content))
	if len(trimmed) == 0 {
		return "新会话"
	}
	if len(trimmed) <= 18 {
		return string(trimmed)
	}
	return string(trimmed[:18]) + "..."
}

func buildPreview(content string) string {
	trimmed := []rune(strings.TrimSpace(content))
	if len(trimmed) <= 36 {
		return string(trimmed)
	}
	return string(trimmed[:36]) + "..."
}

func scanUser(scanner interface{ Scan(...any) error }) (model.User, error) {
	var user model.User
	var createdAt time.Time
	var updatedAt time.Time
	err := scanner.Scan(&user.ID, &user.Account, &user.Username, &user.PasswordHash, &user.Avatar, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}
	user.CreatedAt = nowString(createdAt)
	user.UpdatedAt = nowString(updatedAt)
	return user, nil
}

func scanSession(scanner interface{ Scan(...any) error }) (model.Session, error) {
	var session model.Session
	var createdAt time.Time
	var updatedAt time.Time
	err := scanner.Scan(&session.ID, &session.UserID, &session.Title, &session.LastMessagePreview, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Session{}, ErrSessionNotFound
		}
		return model.Session{}, err
	}
	session.CreatedAt = nowString(createdAt)
	session.UpdatedAt = nowString(updatedAt)
	return session, nil
}

func scanSessionRows(rows *sql.Rows) (model.Session, error) {
	return scanSession(rows)
}

func scanMessageRows(rows *sql.Rows) (model.Message, error) {
	var message model.Message
	var createdAt time.Time
	if err := rows.Scan(&message.ID, &message.SessionID, &message.UserID, &message.Role, &message.Content, &createdAt); err != nil {
		return model.Message{}, err
	}
	message.CreatedAt = nowString(createdAt)
	return message, nil
}
