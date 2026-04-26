package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stock_agent/internal/auth"
	"stock_agent/internal/model"
	"stock_agent/internal/reply"
	"stock_agent/internal/store"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type Router struct {
	repo          store.Repository
	replyService  *reply.Service
	authService   *auth.Service
	sessionMaxAge int
	avatarDir     string
}

type registerRequest struct {
	Account  string `json:"account"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type avatarRequest struct {
	Avatar string `json:"avatar"`
}

type askRequest struct {
	Content string `json:"content"`
}

type askResponse struct {
	Session          model.Session   `json:"session"`
	UserMessage      model.Message   `json:"userMessage"`
	AssistantMessage model.Message   `json:"assistantMessage"`
	Messages         []model.Message `json:"messages"`
}

func New(hostPort string, repo store.Repository, replyService *reply.Service, authService *auth.Service, sessionMaxAge int, avatarDir string) *server.Hertz {
	r := &Router{
		repo:          repo,
		replyService:  replyService,
		authService:   authService,
		sessionMaxAge: sessionMaxAge,
		avatarDir:     avatarDir,
	}

	h := server.Default(server.WithHostPorts(hostPort))

	h.GET("/", r.handleIndex)
	h.GET("/uploads/:file", r.handleAvatarFile)
	h.GET("/api/health", r.handleHealth)
	h.POST("/api/auth/register", r.handleRegister)
	h.POST("/api/auth/login", r.handleLogin)
	h.POST("/api/auth/logout", r.handleLogout)
	h.GET("/api/me", r.handleGetMe)
	h.PUT("/api/me/avatar", r.handleUpdateAvatar)
	h.POST("/api/me/avatar/upload", r.handleUploadAvatar)
	h.POST("/api/sessions", r.handleCreateSession)
	h.GET("/api/sessions", r.handleListSessions)
	h.GET("/api/sessions/:id/messages", r.handleGetMessages)
	h.POST("/api/sessions/:id/messages", r.handleAsk)

	return h
}

func (r *Router) handleIndex(ctx context.Context, c *app.RequestContext) {
	body, err := os.ReadFile("web/index.html")
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to load index page")
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(consts.StatusOK, "text/html; charset=utf-8", body)
}

func (r *Router) handleHealth(ctx context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, utils.H{"status": "ok"})
}

func (r *Router) handleAvatarFile(ctx context.Context, c *app.RequestContext) {
	fileName := filepath.Base(c.Param("file"))
	if fileName == "." || fileName == "" {
		r.writeError(c, consts.StatusNotFound, "file not found")
		return
	}

	fullPath := filepath.Join(r.avatarDir, fileName)
	body, err := os.ReadFile(fullPath)
	if err != nil {
		r.writeError(c, consts.StatusNotFound, "file not found")
		return
	}

	contentType := "application/octet-stream"
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	}

	c.Data(consts.StatusOK, contentType, body)
}

func (r *Router) handleRegister(ctx context.Context, c *app.RequestContext) {
	var req registerRequest
	if err := json.Unmarshal(c.GetRawData(), &req); err != nil {
		r.writeError(c, consts.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Account) == "" || strings.TrimSpace(req.Username) == "" || len(strings.TrimSpace(req.Password)) < 6 {
		r.writeError(c, consts.StatusBadRequest, "账号、用户名和至少 6 位密码是必填项")
		return
	}

	hash, err := r.authService.HashPassword(req.Password)
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to prepare password")
		return
	}

	user, err := r.repo.CreateUser(ctx, req.Account, req.Username, hash, r.authService.DefaultAvatar())
	if err != nil {
		if errors.Is(err, store.ErrAccountExists) {
			r.writeError(c, consts.StatusConflict, "账号已存在")
			return
		}
		r.writeError(c, consts.StatusInternalServerError, "failed to create user")
		return
	}

	if err := r.startSession(ctx, c, user); err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to create login session")
		return
	}

	c.JSON(consts.StatusCreated, utils.H{"user": user})
}

func (r *Router) handleLogin(ctx context.Context, c *app.RequestContext) {
	var req loginRequest
	if err := json.Unmarshal(c.GetRawData(), &req); err != nil {
		r.writeError(c, consts.StatusBadRequest, "invalid json body")
		return
	}

	user, err := r.repo.GetUserByAccount(ctx, req.Account)
	if err != nil {
		r.writeError(c, consts.StatusUnauthorized, "账号或密码错误")
		return
	}
	if !r.authService.VerifyPassword(user.PasswordHash, req.Password) {
		r.writeError(c, consts.StatusUnauthorized, "账号或密码错误")
		return
	}

	if err := r.startSession(ctx, c, user); err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to create login session")
		return
	}

	c.JSON(consts.StatusOK, utils.H{"user": user})
}

func (r *Router) handleLogout(ctx context.Context, c *app.RequestContext) {
	token := r.sessionToken(c)
	if token != "" {
		_ = r.repo.DeleteAuthSession(ctx, token)
	}
	r.clearSessionCookie(c)
	c.JSON(consts.StatusOK, utils.H{"ok": true})
}

func (r *Router) handleGetMe(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}
	c.JSON(consts.StatusOK, utils.H{"user": user})
}

func (r *Router) handleUpdateAvatar(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}

	var req avatarRequest
	if err := json.Unmarshal(c.GetRawData(), &req); err != nil {
		r.writeError(c, consts.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Avatar) == "" {
		r.writeError(c, consts.StatusBadRequest, "avatar is required")
		return
	}

	updatedUser, err := r.repo.UpdateUserAvatar(ctx, user.ID, req.Avatar)
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to update avatar")
		return
	}
	c.JSON(consts.StatusOK, utils.H{"user": updatedUser})
}

func (r *Router) handleUploadAvatar(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		r.writeError(c, consts.StatusBadRequest, "avatar file is required")
		return
	}

	fileName, err := r.storeAvatarFile(fileHeader)
	if err != nil {
		r.writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	updatedUser, err := r.repo.UpdateUserAvatar(ctx, user.ID, "/uploads/"+fileName)
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to update avatar")
		return
	}

	c.JSON(consts.StatusOK, utils.H{"user": updatedUser})
}

func (r *Router) handleCreateSession(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}

	session, err := r.repo.CreateSession(ctx, user.ID)
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to create session")
		return
	}
	c.JSON(consts.StatusCreated, session)
}

func (r *Router) handleListSessions(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}
	sessions, err := r.repo.ListSessions(ctx, user.ID)
	if err != nil {
		r.writeError(c, consts.StatusInternalServerError, "failed to list sessions")
		return
	}
	c.JSON(consts.StatusOK, utils.H{"sessions": sessions})
}

func (r *Router) handleGetMessages(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}

	sessionID := c.Param("id")
	session, err := r.repo.GetSession(ctx, user.ID, sessionID)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}
	messages, err := r.repo.GetMessages(ctx, user.ID, sessionID)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}

	c.JSON(consts.StatusOK, model.SessionMessages{
		Session:  session,
		Messages: messages,
	})
}

func (r *Router) handleAsk(ctx context.Context, c *app.RequestContext) {
	user, ok := r.requireUser(ctx, c)
	if !ok {
		return
	}

	sessionID := c.Param("id")
	var req askRequest
	if err := json.Unmarshal(c.GetRawData(), &req); err != nil {
		r.writeError(c, consts.StatusBadRequest, "invalid json body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		r.writeError(c, consts.StatusBadRequest, "content is required")
		return
	}

	userMessage, _, err := r.repo.AddMessage(ctx, user.ID, sessionID, "user", req.Content)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}
	session, err := r.repo.GetSession(ctx, user.ID, sessionID)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}
	history, err := r.repo.GetMessages(ctx, user.ID, sessionID)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}
	replyContent := r.replyService.GenerateReply(session, history, req.Content)
	assistantMessage, session, err := r.repo.AddMessage(ctx, user.ID, sessionID, "assistant", replyContent)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}
	messages, err := r.repo.GetMessages(ctx, user.ID, sessionID)
	if err != nil {
		r.handleStoreError(c, err)
		return
	}

	c.JSON(consts.StatusOK, askResponse{
		Session:          session,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Messages:         messages,
	})
}

func (r *Router) requireUser(ctx context.Context, c *app.RequestContext) (model.User, bool) {
	token := r.sessionToken(c)
	if token == "" {
		r.writeError(c, consts.StatusUnauthorized, "请先登录")
		return model.User{}, false
	}

	user, err := r.repo.GetUserByToken(ctx, token)
	if err != nil {
		r.writeError(c, consts.StatusUnauthorized, "登录状态已失效，请重新登录")
		return model.User{}, false
	}

	return user, true
}

func (r *Router) startSession(ctx context.Context, c *app.RequestContext, user model.User) error {
	token, err := r.authService.NewSessionToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(time.Duration(r.sessionMaxAge) * time.Second)
	if err := r.repo.CreateAuthSession(ctx, token, user.ID, expiresAt); err != nil {
		return err
	}
	r.setSessionCookie(c, token)
	return nil
}

func (r *Router) sessionToken(c *app.RequestContext) string {
	return string(c.Cookie(auth.SessionCookieName))
}

func (r *Router) setSessionCookie(c *app.RequestContext, token string) {
	c.SetCookie(auth.SessionCookieName, token, r.sessionMaxAge, "/", "", protocol.CookieSameSiteLaxMode, false, true)
}

func (r *Router) clearSessionCookie(c *app.RequestContext) {
	c.SetCookie(auth.SessionCookieName, "", -1, "/", "", protocol.CookieSameSiteLaxMode, false, true)
}

func (r *Router) storeAvatarFile(fileHeader *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	switch ext {
	case ".jpg", ".png", ".webp":
	default:
		return "", errors.New("仅支持 jpg、png、webp 图片")
	}

	if err := os.MkdirAll(r.avatarDir, 0o755); err != nil {
		return "", errors.New("failed to prepare upload directory")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("failed to open upload file")
	}
	defer src.Close()

	fileName, err := store.RandomUploadName("avatar", ext)
	if err != nil {
		return "", errors.New("failed to create file name")
	}

	dstPath := filepath.Join(r.avatarDir, fileName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", errors.New("failed to save upload file")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", errors.New("failed to write upload file")
	}

	return fileName, nil
}

func (r *Router) handleStoreError(c *app.RequestContext, err error) {
	if errors.Is(err, store.ErrSessionNotFound) {
		r.writeError(c, consts.StatusNotFound, "session not found")
		return
	}
	r.writeError(c, consts.StatusInternalServerError, "internal server error")
}

func (r *Router) writeError(c *app.RequestContext, statusCode int, message string) {
	c.JSON(statusCode, utils.H{"error": message})
}

func BuildTestEngine(repo store.Repository, replyService *reply.Service, authService *auth.Service, sessionMaxAge int) *server.Hertz {
	return New(":0", repo, replyService, authService, sessionMaxAge, "uploads")
}
