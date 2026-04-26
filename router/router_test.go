package router

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"stock_agent/internal/auth"
	"stock_agent/internal/reply"
	"stock_agent/internal/store"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRegisterLoginAndProfileFlow(t *testing.T) {
	t.Parallel()

	repo := store.NewMemory()
	h := BuildTestEngine(repo, reply.NewService(), auth.NewService("fern"), 3600)

	registerPayload := []byte(`{"account":"demo_user","username":"演示用户","password":"secret123"}`)
	registerResp := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/auth/register",
		&ut.Body{Body: bytes.NewReader(registerPayload), Len: len(registerPayload)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if registerResp.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", registerResp.Code, registerResp.Body.String())
	}
	cookie := registerResp.Header.Get("Set-Cookie")
	if cookie == "" {
		t.Fatalf("expected login cookie after register")
	}

	meResp := ut.PerformRequest(h.Engine, "GET", "/api/me", nil, ut.Header{Key: "Cookie", Value: cookie})
	if meResp.Code != 200 {
		t.Fatalf("expected 200, got %d", meResp.Code)
	}

	avatarPayload := []byte(`{"avatar":"aurora"}`)
	avatarResp := ut.PerformRequest(
		h.Engine,
		"PUT",
		"/api/me/avatar",
		&ut.Body{Body: bytes.NewReader(avatarPayload), Len: len(avatarPayload)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "Cookie", Value: cookie},
	)
	if avatarResp.Code != 200 {
		t.Fatalf("expected 200, got %d", avatarResp.Code)
	}
}

func TestUserScopedChatFlow(t *testing.T) {
	t.Parallel()

	repo := store.NewMemory()
	h := BuildTestEngine(repo, reply.NewService(), auth.NewService("fern"), 3600)

	aliceCookie := registerUserAndReturnCookie(t, h, "alice", "Alice", "secret123")
	bobCookie := registerUserAndReturnCookie(t, h, "bob", "Bob", "secret123")

	createResp := ut.PerformRequest(h.Engine, "POST", "/api/sessions", nil, ut.Header{Key: "Cookie", Value: aliceCookie})
	if createResp.Code != 201 {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	var session map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	sessionID, _ := session["id"].(string)

	payload := []byte(`{"content":"帮我看看新能源主题基金"}`)
	askResp := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/sessions/"+sessionID+"/messages",
		&ut.Body{Body: bytes.NewReader(payload), Len: len(payload)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "Cookie", Value: aliceCookie},
	)
	if askResp.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", askResp.Code, askResp.Body.String())
	}

	forbiddenResp := ut.PerformRequest(h.Engine, "GET", "/api/sessions/"+sessionID+"/messages", nil, ut.Header{Key: "Cookie", Value: bobCookie})
	if forbiddenResp.Code != 404 {
		t.Fatalf("expected 404, got %d", forbiddenResp.Code)
	}
}

func registerUserAndReturnCookie(t *testing.T, h *server.Hertz, account, username, password string) string {
	t.Helper()

	payload := []byte(`{"account":"` + account + `","username":"` + username + `","password":"` + password + `"}`)
	resp := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/auth/register",
		&ut.Body{Body: bytes.NewReader(payload), Len: len(payload)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if resp.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}
	cookie := resp.Header.Get("Set-Cookie")
	if cookie == "" {
		t.Fatalf("expected Set-Cookie header")
	}
	return strings.Split(cookie, ";")[0]
}
