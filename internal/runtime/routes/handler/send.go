package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	"github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	"github.com/pardnchiu/agenvoy/internal/tools"
	provider "github.com/pardnchiu/go-llm-router/core"
)

type Request struct {
	Content      string   `json:"content"`
	SSE          bool     `json:"sse"`
	SessionID    string   `json:"session_id"`
	Model        string   `json:"model,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`
	Persist      bool     `json:"persist,omitempty"`
	Chat         bool     `json:"chat,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	WorkDir      string   `json:"work_dir,omitempty"`
	Skill        string   `json:"skill,omitempty"`
}

func Send() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
			return
		}

		workDir, err := resolveWorkDir(req.WorkDir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sessionID := req.SessionID
		if sessionID == "" {
			prefix := "temp-"
			switch {
			case req.Chat:
				prefix = "chat-"
			case req.Persist:
				prefix = "http-"
			}
			sessionID = prefix + utils.UUID()
		}

		if exec.IsRunning(sessionID) {
			exec.AppendSteer(sessionID, req.Content)
			c.JSON(http.StatusOK, gin.H{
				"session_id": sessionID,
				"steer":      true,
			})
			return
		}

		events := make(chan agentTypes.Event, 64)
		execCtx := context.WithoutCancel(c.Request.Context())
		wrapped := withFollowup(execCtx, sessionID, pubsub.Wrap(execCtx, sessionID, events, 64))

		go func() {
			defer close(wrapped)

			scanner := agents.Scanner()
			if scanner != nil {
				scanner.Scan()
			}
			trimContent := strings.TrimSpace(req.Content)
			if trimContent != "" {
				wrapped <- agentTypes.Event{Type: agentTypes.EventUserInput, Text: trimContent}
			}

			var matchedSkill *skill.Skill
			var skillResult agentTypes.Event
			if scanner != nil {
				if name := req.Skill; name != "" {
					if slices.Contains(tools.TUIOnlySkills, name) {
						wrapped <- agentTypes.ErrorEvent(fmt.Errorf("skill %q is not available here", name))
						return
					}
					matchedSkill = scanner.Lookup(name)
					if matchedSkill == nil {
						wrapped <- agentTypes.ErrorEvent(fmt.Errorf("skill %q not found", name))
						return
					}
				} else if m, effective := runtime.MatchSkill(scanner, trimContent, tools.TUIOnlySkills...); m != nil {
					matchedSkill = m
					trimContent = strings.TrimSpace(effective)
				}

				if matchedSkill != nil {
					skillResult = agentTypes.Event{Type: agentTypes.EventSkillResult, Text: strings.TrimSpace(matchedSkill.Name)}
					wrapped <- skillResult
					if sessionID != "" {
						sessionLog.Record(sessionID, skillResult)
					}
				}
			}

			userText := trimContent
			if sessionID != "" {
				sessionLog.Append(sessionID, userText)
			}

			wrapped <- agentTypes.Event{Type: agentTypes.EventAgentSelect}
			var agent agentTypes.Agent
			var fallbacks []agentTypes.Agent
			registry := agents.Registry()
			if req.Model != "" {
				if a, ok := registry.Registry[req.Model]; ok {
					agent = a
				}
			}
			if agent == nil {
				primary, rest, err := exec.ResolveAgent(execCtx, agents.DispatcherBot(), registry, trimContent, false, sessionID)
				if err != nil {
					wrapped <- agentTypes.ErrorEvent(err)
					return
				}
				agent = primary
				fallbacks = rest
			}
			agentResult := agentTypes.Event{Type: agentTypes.EventAgentResult, Text: agent.Name()}
			wrapped <- agentResult
			if sessionID != "" {
				sessionLog.Record(sessionID, agentResult)
			}

			data := exec.ExecuteMeta{
				Agent:             agent,
				FallbackAgents:    fallbacks,
				WorkDir:           workDir,
				Skill:             matchedSkill,
				Content:           trimContent,
				Input:             userText,
				ExcludeTools:      append(append([]string{}, tools.TUIOnlyTools...), req.ExcludeTools...),
				ExcludeSkills:     tools.TUIOnlySkills,
				ExtraSystemPrompt: req.SystemPrompt,
			}

			if err := configBot.Save(sessionID, "", "", false); err != nil {
				slog.Warn("sessionBot Save",
					slog.String("session", sessionID),
					slog.String("error", err.Error()))
			}

			session, err := newSession(execCtx, data, sessionID)
			if err != nil {
				wrapped <- agentTypes.ErrorEvent(err)
				return
			}

			if err := exec.Execute(execCtx, data, session, wrapped, data.AllowAll); err != nil {
				wrapped <- agentTypes.ErrorEvent(err)
				return
			}
		}()

		if req.SSE {
			sendSSE(c, sessionID, req.Content, events)
		} else {
			sendResult(c, sessionID, req.Content, events)
		}
		drainEvents(events)
	}
}

func newSession(ctx context.Context, data exec.ExecuteMeta, sessionID string) (*agentTypes.AgentSession, error) {
	session := &agentTypes.AgentSession{
		ID:        sessionID,
		Tools:     []provider.Message{},
		Histories: []provider.Message{},
	}

	scanner := data.SkillScanner
	if scanner == nil {
		scanner = agents.Scanner()
	}
	session.SystemPrompts = exec.BuildSystemPrompts(data.WorkDir, data.ExtraSystemPrompt, scanner, sessionID, data.AllowAll, data.ExcludeSkills)

	oldHistory, maxHistory := sessionHistory.Get(sessionID)
	session.Histories = sessionHistory.Messages(oldHistory)
	session.BaseLen = len(session.Histories)
	session.OldHistories = sessionHistory.Messages(maxHistory)

	if summary := summary.GetPrompt(sessionID, exec.OldestMessageTime(maxHistory)); summary != "" {
		session.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}
	session.ToolHistories = []provider.Message{}

	userText := strings.TrimSpace(data.Input)
	if userText == "" {
		userText = strings.TrimSpace(data.Content)
	}

	session.Sender = data.Sender
	session.UserSendAt = time.Now().UnixNano()
	prefixed := sessionHistory.WithPrefix(sessionHistory.Record{
		SendAt: session.UserSendAt,
		Sender: session.Sender,
	}.Prefix(), userText)

	session.UserInput = provider.Message{Role: "user", Content: prefixed}
	session.Histories = append(session.Histories, provider.Message{
		Role:    "user",
		Content: prefixed,
	})
	exec.SaveUserInputHistory(ctx, sessionID, userText)

	return session, nil
}

func resolveWorkDir(input string) (string, error) {
	home, _ := os.UserHomeDir()
	dir := strings.TrimSpace(input)
	if dir == "" {
		return home, nil
	}

	resolved, err := go_pkg_filesystem.AbsPath(home, dir, go_pkg_filesystem.AbsPathOption{})
	if err != nil {
		return "", fmt.Errorf("work_dir %q cannot be resolved: %w", dir, err)
	}
	if !go_pkg_filesystem_reader.Exists(resolved) {
		return "", fmt.Errorf("work_dir %q does not exist", dir)
	}
	if !go_pkg_filesystem_reader.IsDir(resolved) {
		return "", fmt.Errorf("work_dir %q is not a directory", dir)
	}
	return resolved, nil
}
