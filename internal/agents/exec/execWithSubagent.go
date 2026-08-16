package exec

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	"github.com/pardnchiu/agenvoy/internal/tools"
)

const maxConcurrentSubagents = 3

var subagentSlots = make(chan struct{}, maxConcurrentSubagents)

func ExecWithSubagent(ctx context.Context, task, sessionIDInput, model, reasoning, systemPrompt string, excludedTools []string, parentSessionID string) (string, error) {
	registry := agents.Registry()
	dispatcher := agents.DispatcherBot()
	if dispatcher == nil || len(registry.Registry) == 0 {
		return "", fmt.Errorf("subagent host not initialized")
	}

	select {
	case subagentSlots <- struct{}{}:
		defer func() { <-subagentSlots }()
	case <-ctx.Done():
		return "", fmt.Errorf("waiting for a subagent slot: %w", ctx.Err())
	}

	sessionID, err := ensureSubagentSession(sessionIDInput)
	if err != nil {
		return "", fmt.Errorf("ensureSubagentSession: %w", err)
	}

	if strings.TrimSpace(sessionIDInput) != "" && !strings.HasPrefix(sessionID, "temp-") {
		sessionModel, sessionReasoning := configBot.GetModel(sessionID)
		model = ""
		if sessionModel != configBot.DefaultModel {
			model = sessionModel
		}
		reasoning = sessionReasoning
	}

	var agent agentTypes.Agent
	if model != "" {
		agent = registry.Registry[model]
	} else {
		agent = SelectAgent(ctx, dispatcher, registry, task, false, "")
	}
	if agent == nil {
		return "", fmt.Errorf("no agent available")
	}

	allowAll, ok := ctx.Value(allowAllCtxKey{}).(bool)
	if !ok {
		allowAll = true
	}

	workDir, ok := ctx.Value(parentWorkDirKey{}).(string)
	if !ok || workDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workDir = cwd
		} else if home, err := os.UserHomeDir(); err == nil {
			workDir = home
		} else {
			return "", fmt.Errorf("cwd and home both failed")
		}
	}
	// * collection-only charter: no nesting, no writes, no deliverable renderers
	subagentExcludeBase := []string{
		"invoke_subagent", "list_subagent_sessions",
		"write_file", "patch_file", "generate*",
	}
	excluded := append(append(subagentExcludeBase, tools.TUIOnlyTools...), excludedTools...)

	charter := configs.SubagentCharter
	if extra := strings.TrimSpace(systemPrompt); extra != "" {
		charter += "\n\n---\n\n" + extra
	}
	execData := ExecuteMeta{
		Agent:             agent,
		WorkDir:           workDir,
		Content:           task,
		ExcludeTools:      excluded,
		ExcludeSkills:     tools.TUIOnlySkills,
		ExtraSystemPrompt: charter,
		Reasoning:         reasoning,
		AllowAll:          allowAll,
	}

	oldRecords, maxRecords := sessionHistory.Get(sessionID)
	oldHistory := sessionHistory.Messages(oldRecords)
	maxHistory := sessionHistory.Messages(maxRecords)

	sendAt := time.Now().UnixNano()
	userText := task
	prefixed := sessionHistory.WithPrefix(sessionHistory.Record{SendAt: sendAt}.Prefix(), userText)

	histories := append([]provider.Message{}, oldHistory...)
	histories = append(histories, provider.Message{Role: "user", Content: prefixed})

	session := &agentTypes.AgentSession{
		ID:            sessionID,
		SystemPrompts: BuildSystemPrompts(execData.WorkDir, execData.ExtraSystemPrompt, agents.Scanner(), sessionID, execData.AllowAll, execData.ExcludeSkills),
		OldHistories:  maxHistory,
		ToolHistories: []provider.Message{},
		Tools:         []provider.Message{},
		Histories:     histories,
		BaseLen:       len(oldHistory),
		UserSendAt:    sendAt,
		UserInput:     provider.Message{Role: "user", Content: prefixed},
	}
	if summary := summary.GetPrompt(sessionID, OldestMessageTime(maxRecords)); summary != "" {
		session.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	sessionLog.Append(sessionID, userText)
	SaveUserInputHistory(ctx, sessionID, userText)

	subCtx, cancel := context.WithTimeout(ctx, time.Duration(filesystem.MaxSubagentTimeoutMin)*time.Minute)
	defer cancel()

	parentEvents, ok := ctx.Value(parentEventsKey{}).(chan<- agentTypes.Event)
	if !ok {
		parentEvents = nil
	}

	displayName, _ := configBot.Get(sessionID)
	if displayName == "" || displayName == sessionID {
		var short, rest string
		switch {
		case strings.HasPrefix(sessionID, "temp-"):
			short, rest = "temp-", sessionID[len("temp-"):]
		case strings.HasPrefix(sessionID, "cli-"):
			short, rest = "cli-", sessionID[len("cli-"):]
		case strings.HasPrefix(sessionID, "http-"):
			short, rest = "http-", sessionID[len("http-"):]
		}
		if short != "" {
			if len(rest) > 8 {
				rest = rest[:8]
			}
			displayName = short + rest
		}
	}

	events := make(chan agentTypes.Event, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(events)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("subagent Execute panic recovered",
					slog.String("session", sessionID),
					slog.Any("panic", r))
				errCh <- fmt.Errorf("subagent execute panicked: %v", r)
			}
		}()
		errCh <- Execute(subCtx, execData, session, events, allowAll)
	}()

	var sb strings.Builder
	var totalUsage provider.Usage
	for ev := range events {
		pubsub.Pub(sessionID, ev)
		passSubagentEvent(parentEvents, displayName, ev)

		switch ev.Type {
		case agentTypes.EventText:
			if ev.Text == "" {
				continue
			}
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(ev.Text)
		case agentTypes.EventDone:
			if ev.Usage != nil {
				totalUsage.Input += ev.Usage.Input
				totalUsage.Output += ev.Usage.Output
				totalUsage.CacheCreate += ev.Usage.CacheCreate
				totalUsage.CacheRead += ev.Usage.CacheRead
			}
		case agentTypes.EventError:
			if ev.Err != nil {
				slog.Warn("subagent event error",
					slog.String("session", sessionID),
					slog.String("error", ev.Err.Error()))
			}
		}
	}

	passSubagentEvent(parentEvents, displayName, agentTypes.Event{
		Type:     agentTypes.EventToolResult,
		ToolName: "invoke_subagent",
	})

	usageLine := fmt.Sprintf("usage: in=%d out=%d cached=%d write=%d", totalUsage.Input+totalUsage.CacheRead+totalUsage.CacheCreate, totalUsage.Output, totalUsage.CacheRead, totalUsage.CacheCreate)

	if parentSessionID != "" && parentSessionID != sessionID && (totalUsage.Input > 0 || totalUsage.Output > 0 || totalUsage.CacheRead > 0 || totalUsage.CacheCreate > 0) {
		prov, usageModel, _ := strings.Cut(agent.Name(), "@")
		usagelog.Append(parentSessionID, prov, usageModel, totalUsage)
	}

	retryHint := ""
	if ctx.Err() == nil {
		retryHint = fmt.Sprintf(" Re-dispatch this leg with a model other than %s; the rest of the fan-out is unaffected.", agent.Name())
	}

	if err := <-errCh; err != nil {
		if str := strings.TrimSpace(sb.String()); str != "" {
			slog.Warn("subagent partial output discarded",
				slog.String("session", sessionID),
				slog.String("model", agent.Name()),
				slog.String("output", go_pkg_utils.TruncateString(str, 2048)))
		}
		return "", fmt.Errorf("subagent %s failed: %w.%s", agent.Name(), err, retryHint)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("subagent %s finished without producing any text (%s).%s",
			agent.Name(), usageLine, retryHint)
	}
	return fmt.Sprintf("[subagent · %s · session=%s · %s]\n%s", agent.Name(), sessionID, usageLine, result), nil
}

func passSubagentEvent(parent chan<- agentTypes.Event, name string, ev agentTypes.Event) {
	if parent == nil {
		return
	}
	switch ev.Type {
	case agentTypes.EventDone,
		agentTypes.EventText,
		agentTypes.EventTextDone,
		agentTypes.EventAgentSelect,
		agentTypes.EventAgentResult,
		agentTypes.EventSummaryGenerate,
		agentTypes.EventToolCallStart,
		agentTypes.EventToolCallEnd,
		agentTypes.EventToolCallText,
		agentTypes.EventSkillResult,
		agentTypes.EventTodoUpdate:
		return
	}

	out := ev
	if out.Source == "" {
		out.Source = name
	}
	parent <- out
}

func ensureSubagentSession(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		if idle := sessionManager.FindIdleTemp(); idle != "" {
			if _, err := sessionManager.ResetAll(idle); err != nil {
				slog.Warn("ensureSubagentSession ResetAll, opening a fresh session instead",
					slog.String("session", idle),
					slog.String("error", err.Error()))
			} else {
				return idle, nil
			}
		}
		id, err := sessionManager.New("temp-")
		if err != nil {
			return "", fmt.Errorf("sessionManager.CreateSession: %w", err)
		}
		return id, nil
	}

	sessionDir := filesystem.SessionDir(trimmed)
	if !go_pkg_filesystem_reader.Exists(sessionDir) {
		return "", fmt.Errorf("session %q does not exist", trimmed)
	}
	if !go_pkg_filesystem_reader.IsDir(sessionDir) {
		return "", fmt.Errorf("session %q is not a directory", trimmed)
	}
	return trimmed, nil
}
