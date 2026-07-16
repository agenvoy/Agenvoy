package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	go_bot_discord "github.com/pardnchiu/go-bot/discord"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionDiscord "github.com/pardnchiu/agenvoy/internal/session/discord"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
)

func getSession(ctx context.Context, in go_bot_discord.Input, content string, data exec.ExecData) (*agentTypes.AgentSession, error) {
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
	sess.Histories = oldHistory
	sess.BaseLen = len(oldHistory)

	sess.SystemPrompts = exec.BuildSystemPrompts(data.WorkDir, data.ExtraSystemPrompt, agents.Scanner(), sessionID, data.AllowAll, data.ExcludeSkills)
	if summary := summary.GetPrompt(sessionID, exec.OldestMessageTime(maxHistory)); summary != "" {
		sess.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	sess.OldHistories = maxHistory
	sess.ToolHistories = []provider.Message{}

	userText := strings.TrimSpace(data.Input)
	if userText == "" {
		header := fmt.Sprintf("當前時間: %s\n工作目錄: %s\n傳送者: %s\n當前 channel: %s",
			time.Now().Format("2006-01-02 15:04:05"),
			data.WorkDir,
			in.Username,
			channelName(in),
		)
		userText = fmt.Sprintf("---\n%s\n---\n%s", header, strings.TrimSpace(content))
	}

	histText := userText
	if h := strings.TrimSpace(data.HistoryContent); h != "" {
		histText = h
	}
	sess.Histories = append(sess.Histories, provider.Message{
		Role:    "user",
		Content: histText,
	})
	sess.UserInput = provider.Message{
		Role:    "user",
		Content: userText,
	}
	exec.SaveUserInputHistory(ctx, sessionID, histText)

	return sess, nil
}
