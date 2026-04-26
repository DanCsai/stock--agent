package reply

import (
	"fmt"
	"strings"

	"stock_agent/internal/model"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GenerateReply(session model.Session, history []model.Message, userMessage string) string {
	recentTopic := ""
	userQuestions := 0

	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != "user" {
			continue
		}
		userQuestions++
		if strings.TrimSpace(msg.Content) != "" && recentTopic == "" && msg.Content != userMessage {
			recentTopic = trimPreview(msg.Content, 24)
		}
	}

	if recentTopic == "" {
		return fmt.Sprintf("基金助手 MVP 已收到你的问题：%q。当前版本先完成前端界面、接口和会话历史能力，后续会继续补充模型分析与推荐能力。", userMessage)
	}

	return fmt.Sprintf("基金助手 MVP 已收到你的问题：%q。我们会沿着你刚才提到的“%s”继续保留上下文；当前这是占位回复，用于验证聊天流程、历史会话和接口联调。你在这个会话里已经提了 %d 个问题。", userMessage, recentTopic, userQuestions)
}

func trimPreview(input string, limit int) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}
