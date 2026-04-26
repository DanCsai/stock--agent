package model

type User struct {
	ID           int64  `json:"id"`
	Account      string `json:"account"`
	Username     string `json:"username"`
	Avatar       string `json:"avatar"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type Session struct {
	ID                 string `json:"id"`
	UserID             int64  `json:"-"`
	Title              string `json:"title"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	LastMessagePreview string `json:"lastMessagePreview"`
}

type Message struct {
	ID        string `json:"id"`
	UserID    int64  `json:"-"`
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type SessionMessages struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}
