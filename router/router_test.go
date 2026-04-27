package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"stock_agent/internal/auth"
	"stock_agent/internal/fund"
	"stock_agent/internal/reply"
	"stock_agent/internal/store"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRegisterLoginAndProfileFlow(t *testing.T) {
	t.Parallel()

	repo := store.NewMemory()
	h := BuildTestEngine(repo, reply.NewService(), nil, auth.NewService("fern"), 3600)

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
	h := BuildTestEngine(repo, reply.NewService(), nil, auth.NewService("fern"), 3600)

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

func TestFundQueryChatAndDetailAPI(t *testing.T) {
	t.Parallel()

	repo := store.NewMemory()
	fundService := fund.NewService(stubFundProvider{})
	h := BuildTestEngine(repo, reply.NewService(), fundService, auth.NewService("fern"), 3600)

	cookie := registerUserAndReturnCookie(t, h, "fund_user", "Fund User", "secret123")
	createResp := ut.PerformRequest(h.Engine, "POST", "/api/sessions", nil, ut.Header{Key: "Cookie", Value: cookie})
	if createResp.Code != 201 {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	var session map[string]any
	if err := json.Unmarshal(createResp.Body.Bytes(), &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	sessionID, _ := session["id"].(string)

	payload := []byte(`{"content":"帮我看看000001"}`)
	askResp := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/sessions/"+sessionID+"/messages",
		&ut.Body{Body: bytes.NewReader(payload), Len: len(payload)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "Cookie", Value: cookie},
	)
	if askResp.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", askResp.Code, askResp.Body.String())
	}
	if !strings.Contains(askResp.Body.String(), `"fundResult"`) {
		t.Fatalf("expected fundResult in response body=%s", askResp.Body.String())
	}

	detailResp := ut.PerformRequest(h.Engine, "GET", "/api/funds/000001?range=3m", nil, ut.Header{Key: "Cookie", Value: cookie})
	if detailResp.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", detailResp.Code, detailResp.Body.String())
	}
	if !strings.Contains(detailResp.Body.String(), `"range":"3m"`) {
		t.Fatalf("expected 3m range in response body=%s", detailResp.Body.String())
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

type stubFundProvider struct{}

func (stubFundProvider) Search(_ context.Context, query string, limit int) ([]fund.SearchCandidate, error) {
	if strings.Contains(query, "000001") || strings.Contains(query, "华夏") {
		return []fund.SearchCandidate{
			{Code: "000001", Name: "华夏成长", Type: "混合型"},
		}, nil
	}
	return nil, nil
}

func (stubFundProvider) GetProfile(_ context.Context, code string) (*fund.Profile, error) {
	if code != "000001" {
		return nil, errors.New("not found")
	}
	return &fund.Profile{
		Code:        "000001",
		Name:        "华夏成长",
		Type:        "混合型",
		Company:     "华夏基金",
		LatestNAV:   "1.2345",
		DailyChange: "0.82",
		AsOfDate:    "2026-04-27",
	}, nil
}

func (stubFundProvider) GetTrend(_ context.Context, code string, rangeKey string) (*fund.Trend, error) {
	if code != "000001" {
		return nil, errors.New("not found")
	}
	return &fund.Trend{
		Range: rangeKey,
		Points: []fund.TrendPoint{
			{Date: "2026-04-01", Value: 1.12},
			{Date: "2026-04-15", Value: 1.18},
			{Date: "2026-04-27", Value: 1.23},
		},
	}, nil
}
