package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	go_bot_discord "github.com/pardnchiu/go-bot/discord"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionDiscord "github.com/pardnchiu/agenvoy/internal/session/discord"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func getSession(ctx context.Context, in go_bot_discord.Input, content string, data exec.ExecuteMeta) (*agentTypes.AgentSession, error) {
	sessionID, err := sessionDiscord.New(in.GuildID, in.ChannelID, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetDiscordSession: %w", err)
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

	histText := userText
	if h := strings.TrimSpace(data.HistoryContent); h != "" {
		histText = h
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
		Content: sessionHistory.WithPrefix(prefix, histText),
	})
	sess.UserInput = provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, userText),
	}
	exec.SaveUserInputHistory(ctx, sessionID, histText)

	return sess, nil
}
