package subagent

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
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/model"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	"github.com/pardnchiu/agenvoy/internal/tools"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
)

var channLen = make(chan struct{}, min(configs.MAX_CONCURRENT_TOOLS, configs.MAX_CONCURRENT_SUBAGENTS))

func Exec(ctx context.Context, task, sessionID, modelName, reasoning, systemPrompt string, excludedTools []string, ignoreHistory bool) (string, error) {
	// * step 1: model exist
	registry := agents.ModelSet()
	if len(registry.Registry) == 0 {
		return "", fmt.Errorf("no model is registered; subagents are unavailable")
	}

	// * step 2: concurrent limit
	// * prepare channel to limit concurrent tasks, if full, then retry with a new subagent
	select {
	case channLen <- struct{}{}:
		defer func() { <-channLen }()
	case <-ctx.Done():
		return "", fmt.Errorf("subagent slots are full; call a new subagent to retry this task: %w", ctx.Err())
	}

	// * step 3: session validation and history
	sessID, err := ensureSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("subagent session: %w", err)
	}
	var maxRec []history.Record
	// * if subagent call with temp-, call it as a new session
	if strings.HasPrefix(sessID, "temp-") {
		ignoreHistory = true
	} else if !ignoreHistory {
		_, maxRec = history.Get(sessID)
	}
	contextHistories := history.Messages(maxRec)

	// * step 4: model/reasoning validation
	// * if session is not temp, then use default configured model and reasoning
	if !strings.HasPrefix(sessID, "temp-") {
		sessModel, sessReasoning := configBot.GetModel(sessID)
		// * only if not auto, then replace, or using input model / reasoning
		if sessModel != configBot.DefaultModel {
			modelName = sessModel
		}
		if sessReasoning != configs.REASONING_AUTO {
			reasoning = sessReasoning
		}
	}

	trimTask := strings.TrimSpace(task)
	var agent agenttypes.Agent
	// * if model is auto, then select model first
	if modelName == configBot.DefaultModel {
		agent = model.Select(ctx, model.SelectMeta{
			Agent:     agents.DispatcherBot(),
			ModelSet:  registry,
			Task:      trimTask,
			SessionID: sessID,
		})
	} else {
		agent = registry.Registry[modelName]
	}
	if agent == nil {
		return "", fmt.Errorf("model %q is not registered; use another model", modelName)
	}

	// * step 5: working directory
	workDir, ok := ctx.Value(exec.ParentWorkDirKey{}).(string)
	if !ok || workDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workDir = cwd
		} else if home, err := os.UserHomeDir(); err == nil {
			workDir = home
		} else {
			return "", fmt.Errorf("no work directory: cwd and home are both unreadable")
		}
	}

	// * step 6: allowAll flag
	allowAll, ok := ctx.Value(exec.AllowAllCtxKey{}).(bool)
	if !ok {
		allowAll = true
	}

	// * step 7: excluded tools
	// * collection-only, no nesting, no writes, no deliverable renderers
	toolExclude := append(append([]string{
		"subagents",
		"edit_file",
		"edit_tool",
		"test_tool",
		"edit_skill",
		"write_result",
		"html_template",
		"generate*",
		"open_file",
		"schedules",
		"store_secret",
		"pkg_manage",
	}, tools.TUIOnlyTools...), excludedTools...)

	// * step 8: create task
	trimSystemPrompt := strings.TrimSpace(systemPrompt)
	if trimSystemPrompt != "" {
		trimSystemPrompt = "---\n\n" + trimSystemPrompt
	}
	charter := strings.TrimSpace(strings.ReplaceAll(configs.SubagentPrompt, "{{.SystemPrompt}}", trimSystemPrompt))
	hash := interactive.CreateExecPending(sessID, "", trimTask, "", agent.Name(), reasoning, allowAll)
	hashPath := filesystem.PendingMetaPath(sessID, hash)
	taskMeta := exec.ExecuteMeta{
		Agent:             agent,
		WorkDir:           workDir,
		PendingTask:       hash,
		KeepPending:       true,
		IgnoreHistory:     ignoreHistory,
		Task:              trimTask,
		ExcludeTools:      toolExclude,
		ExcludeSkills:     tools.TUIOnlySkills,
		ExtraSystemPrompt: charter,
		Reasoning:         reasoning,
		AllowAll:          allowAll,
	}

	// * step 9: prepare session
	sendAt := time.Now().UnixNano()
	prefixed := history.WithPrefix(history.Record{SendAt: sendAt}.Prefix(), trimTask)
	session := &agenttypes.Session{
		ID:               sessID,
		SystemPrompts:    exec.BuildSystemPrompts(taskMeta.WorkDir, taskMeta.ExtraSystemPrompt, sessID, taskMeta.Agent.Name()),
		ContextHistories: contextHistories,
		ToolHistories:    []provider.Message{},
		ToolResults:      []provider.Message{},
		RawHistories:     []provider.Message{{Role: "user", Content: prefixed}},
		SendAt:           sendAt,
		UserMessage:      provider.Message{Role: "user", Content: prefixed},
	}
	if !ignoreHistory {
		if summary := summary.GetPrompt(sessID, exec.GetOldestMessageTime(maxRec)); summary != "" {
			session.Summary = provider.Message{Role: "user", Content: summary}
		}
	}

	// * step 10: execute
	if exec.IsSchedule(ctx) {
		// * if is schedule jsut record result, no task content
		sessionLog.Append(sessID, agenttypes.WindowHash(ctx), "[Scheduled Task: "+exec.DCPushPrefix(ctx)+"]")
	} else {
		pubsub.Pub(sessID, agenttypes.Event{Type: agenttypes.EventUserInput, Text: trimTask})
		sessionLog.Append(sessID, agenttypes.WindowHash(ctx), trimTask)
	}
	exec.SaveUserInputHistory(ctx, sessID, trimTask)

	// * step 11: execute with timeout
	parentSessID := agenttypes.DeliverToFrom(ctx)
	if parentSessID == "" {
		parentSessID = agenttypes.SessionIDFrom(ctx)
	}
	subCtx, cancel := context.WithTimeout(agenttypes.WithDeliverTo(ctx, parentSessID), time.Duration(configs.MAX_SUBAGENT_TIMEOUT_MIN)*time.Minute)
	defer cancel()

	parentEvents, ok := ctx.Value(exec.ParentEventsKey{}).(chan<- agenttypes.Event)
	if !ok {
		parentEvents = nil
	}

	// * step 12: agent selected event
	agentSelectedEvent := agenttypes.Event{Type: agenttypes.EventAgentResult, Text: agent.Name(), Reasoning: exec.ResolveReasoning(sessID, reasoning).String()}
	if exec.IsSchedule(ctx) {
		agentSelectedEvent.Text = exec.ScheduleLabel(exec.DCPushPrefix(ctx))
	}
	pubsub.Pub(sessID, agentSelectedEvent)
	sessionLog.Record(sessID, agentSelectedEvent)

	// * step 13: execute in goroutine
	events := make(chan agenttypes.Event, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(events)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("subagent Execute panic recovered",
					slog.String("session", sessID),
					slog.Any("panic", r))
				errCh <- fmt.Errorf("subagent crashed: %v", r)
			}
		}()
		errCh <- exec.Execute(subCtx, taskMeta, session, events, allowAll)
	}()

	// * step 14: collect events
	displayName, _ := configBot.Get(sessID)
	if displayName == "" || displayName == sessID {
		for _, prefix := range []string{"temp-", "cli-", "dc-", "tg-"} {
			if rest, ok := strings.CutPrefix(sessID, prefix); ok {
				displayName = prefix + rest[:min(len(rest), 8)]
				break
			}
		}
	}

	var (
		sb    strings.Builder
		usage provider.Usage
	)
	for ev := range events {
		pubsub.Pub(sessID, ev)
		sendEvent(parentEvents, displayName, ev)

		switch ev.Type {
		case agenttypes.EventText:
			if ev.Text == "" {
				continue
			}
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(ev.Text)
		case agenttypes.EventDone:
			if ev.Usage != nil {
				usage.Input += ev.Usage.Input
				usage.Output += ev.Usage.Output
				usage.CacheCreate += ev.Usage.CacheCreate
				usage.CacheRead += ev.Usage.CacheRead
			}
		case agenttypes.EventError:
			if ev.Err != nil {
				slog.Debug("subagent event error",
					slog.String("session", sessID),
					slog.String("error", ev.Err.Error()))
			}
		}
	}

	sendEvent(parentEvents, displayName, agenttypes.Event{
		Type:     agenttypes.EventToolResult,
		ToolName: "subagents",
	})

	formatUsage := func(usage provider.Usage) string {
		return fmt.Sprintf("usage: in=%d out=%d cached=%d write=%d", usage.Input+usage.CacheRead+usage.CacheCreate, usage.Output, usage.CacheRead, usage.CacheCreate)
	}
	usageText := formatUsage(usage)
	retryMessage := ""
	if ctx.Err() == nil {
		retryMessage = fmt.Sprintf(" Call a new subagent to retry only this task, with a model other than %s.", agent.Name())
	}

	execErr := <-errCh
	result := strings.TrimSpace(sb.String())
	if execErr == nil && result == "" {
		if info, ok := interactive.LoadPendingInfo(sessID, hash); ok && info.HasQuestions {
			answered, resumeUsage := waitResume(subCtx, sessID, parentEvents, displayName)
			if answered != "" {
				result = answered
				usage.Input += resumeUsage.Input
				usage.Output += resumeUsage.Output
				usage.CacheCreate += resumeUsage.CacheCreate
				usage.CacheRead += resumeUsage.CacheRead
				usageText = formatUsage(usage)
			}
		}
	}

	if execErr == nil && result != "" {
		interactive.CleanupPending(sessID, hash)
		return fmt.Sprintf("[subagent · %s · session=%s · %s]\n%s", agent.Name(), sessID, usageText, result), nil
	}

	interactive.CompactPending(sessID, hash)
	partialHint := ""
	if go_pkg_filesystem_reader.Exists(hashPath) {
		partialHint = fmt.Sprintf(" Partial results are saved at %s; have the new subagent read_files it first and continue from there.", hashPath)
	}

	if execErr != nil {
		if result != "" {
			slog.Debug("subagent partial output discarded",
				slog.String("session", sessID),
				slog.String("model", agent.Name()),
				slog.String("output", go_pkg_utils.TruncateString(result, 2048)))
		}
		return "", fmt.Errorf("subagent %s failed: %w.%s%s", agent.Name(), execErr, retryMessage, partialHint)
	}
	return "", fmt.Errorf("subagent %s returned no text (%s).%s%s", agent.Name(), usageText, retryMessage, partialHint)
}
