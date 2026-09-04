package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	sessionTelegram "github.com/pardnchiu/agenvoy/internal/session/telegram"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func getSession(ctx context.Context, chatID int64, username, content string, data exec.ExecuteMeta, overrideID string) (*agentTypes.AgentSession, error) {
	chatSessionID, err := sessionTelegram.New(chatID)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetTelegramSession: %w", err)
	}

	histSessionID := chatSessionID
	if id := strings.TrimSpace(overrideID); id != "" {
		histSessionID = id
	}

	sess := &agentTypes.AgentSession{
		ID:        histSessionID,
		Tools:     []provider.Message{},
		Histories: []provider.Message{},
	}

	oldHistory, maxHistory := sessionHistory.Get(histSessionID)
	sess.Histories = sessionHistory.Messages(oldHistory)
	sess.BaseLen = len(sess.Histories)

	sess.SystemPrompts = exec.BuildSystemPrompts(data.WorkDir, data.ExtraSystemPrompt, agents.Scanner(), chatSessionID, data.AllowAll, data.ExcludeSkills, data.ModelName())
	if summary := summary.GetPrompt(histSessionID, exec.OldestMessageTime(maxHistory)); summary != "" {
		sess.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	sess.OldHistories = sessionHistory.Messages(maxHistory)
	sess.ToolHistories = []provider.Message{}

	userText := strings.TrimSpace(data.Input)
	if userText == "" {
		userText = strings.TrimSpace(content)
	}

	histText := userText
	if h := strings.TrimSpace(data.HistoryContent); h != "" {
		histText = h
	}

	sess.Sender = strings.TrimSpace(username)
	if sess.Sender == "" {
		sess.Sender = strings.TrimSpace(data.Sender)
	}
	sess.UserSendAt = time.Now().UnixNano()
	prefix := sessionHistory.Record{
		SendAt: sess.UserSendAt,
		Sender: sess.Sender,
	}.Prefix()

	sess.Histories = append(sess.Histories, provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, histText),
	})
	sess.UserInput = provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, userText),
	}
	exec.SaveUserInputHistory(ctx, histSessionID, histText)

	return sess, nil
}
