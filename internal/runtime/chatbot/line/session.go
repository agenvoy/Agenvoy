package line

import (
	"context"
	"fmt"
	"strings"
	"time"

	go_bot_line "github.com/pardnchiu/go-bot/line"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func getSession(ctx context.Context, in go_bot_line.Input, content string, data exec.ExecuteMeta) (*agentTypes.AgentSession, error) {
	sessionID, err := sessionManager.GetLineSession(in.UserID, in.GroupID, in.RoomID)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetLineSession: %w", err)
	}

	sess := &agentTypes.AgentSession{
		ID:        sessionID,
		Tools:     []provider.Message{},
		Histories: []provider.Message{},
	}

	oldHistory, maxHistory := sessionHistory.Get(sessionID)
	sess.Histories = sessionHistory.Messages(oldHistory)
	sess.BaseLen = len(sess.Histories)

	sess.SystemPrompts = exec.BuildSystemPrompts(data.WorkDir, data.ExtraSystemPrompt, agents.Scanner(), sessionID, data.AllowAll, data.ExcludeSkills)
	if summary := summary.GetPrompt(sessionID, exec.OldestMessageTime(maxHistory)); summary != "" {
		sess.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	sess.OldHistories = sessionHistory.Messages(maxHistory)
	sess.ToolHistories = []provider.Message{}

	userText := strings.TrimSpace(data.Input)
	if userText == "" {
		userText = strings.TrimSpace(content)
	}

	sess.Sender = strings.TrimSpace(in.Username)
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
		Content: sessionHistory.WithPrefix(prefix, userText),
	})
	sess.UserInput = provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, userText),
	}
	sessionLog.Append(sessionID, userText)
	exec.SaveUserInputHistory(ctx, sessionID, userText)

	return sess, nil
}
