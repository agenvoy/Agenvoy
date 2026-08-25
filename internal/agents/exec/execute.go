package exec

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	go_pkg_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	allowSkill "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/skill"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/compact"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/cooldown"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	"github.com/pardnchiu/agenvoy/internal/tools"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

type ExecuteMeta struct {
	Agent             agentTypes.Agent
	FallbackAgents    []agentTypes.Agent
	WorkDir           string
	Skill             *skill.Skill
	SkillScanner      *runtime.SkillScanner
	Content           string
	Input             string
	SessionID         string
	ImageInputs       []string
	FileInputs        []string
	ExcludeTools      []string
	ExcludeSkills     []string
	ClientTools       []provider.Tool
	ExtraSystemPrompt string
	Reasoning         string
	AllowAll          bool
	PendingTask       string
	ReplyMessageID    string
	HistoryContent    string
	Sender            string
}

type (
	allowAllCtxKey   struct{}
	parentEventsKey  struct{}
	parentWorkDirKey struct{}
)

func Execute(ctx context.Context, data ExecuteMeta, session *agentTypes.AgentSession, events chan<- agentTypes.Event, allowAll bool) error {
	execCtx := agentTypes.WithSessionID(ctx, session.ID)
	execStart := time.Now()

	if !allowAll {
		if data.Skill != nil && strings.TrimSpace(data.Skill.Content) != "" && allowSkill.Match(data.WorkDir, data.Skill.Name) {
			allowAll = true
		}
	}

	execCtx = context.WithValue(execCtx, allowAllCtxKey{}, allowAll)

	if events != nil {
		execCtx = context.WithValue(execCtx, parentEventsKey{}, events)
	}

	if strings.TrimSpace(data.WorkDir) != "" {
		execCtx = context.WithValue(execCtx, parentWorkDirKey{}, data.WorkDir)
	}

	pushCtx := execCtx
	execCtx, execCancel := context.WithCancel(execCtx)
	defer execCancel()

	var taskID string
	if session.ID != "" {
		taskID = go_pkg_utils.UUID()
		configStatus.Online(session.ID)
		defer configStatus.Idle(session.ID)
		registerCancel(taskID, execCancel)
		defer unregisterCancel(taskID)

		if err := sessionManager.AddConcurrent(execCtx, session.ID); err != nil {
			return fmt.Errorf("EnterConcurrent: %w", err)
		}
		defer sessionManager.RemoveConcurrent(session.ID)
		defer markRunning(session.ID)()
		defer ClearSteer(session.ID)

		original := events
		runTaskID := taskID
		fanoutEvents := make(chan agentTypes.Event, 64)
		done := make(chan struct{})
		sid := session.ID
		pushHook, hasPush := lookupPushHook(sid)
		isDcPush := hasPush && !isDcPushSuppressed(pushCtx)
		var pushTextBuf strings.Builder
		var pushDoneEv agentTypes.Event
		stateless := session.Stateless
		go func() {
			defer close(done)
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event fanout goroutine panic recovered",
						slog.String("session", sid),
						slog.Any("panic", r))
				}
			}()
			for ev := range fanoutEvents {
				if ev.TaskID == "" && ev.Source == "" {
					ev.TaskID = runTaskID
				}
				if !stateless && ev.Source == "" {
					sessionLog.Record(sid, ev)
				}
				if isDcPush {
					switch ev.Type {
					case agentTypes.EventText:
						if ev.Source == "" && ev.Text != "" {
							if pushTextBuf.Len() > 0 {
								pushTextBuf.WriteByte('\n')
							}
							pushTextBuf.WriteString(ev.Text)
						}
					case agentTypes.EventDone:
						if ev.Source == "" {
							pushDoneEv = ev
						}
					}
				}
				original <- ev
			}
		}()
		defer func() {
			close(fanoutEvents)
			<-done
			if isDcPush {
				text := strings.TrimSpace(pushTextBuf.String())
				if text != "" {
					pushHook(pushCtx, PushPayload{
						SessionID: sid,
						Text:      text,
						Model:     pushDoneEv.Model,
						Usage:     pushDoneEv.Usage,
						Duration:  pushDoneEv.Duration,
						Prefix:    dcPushPrefix(pushCtx),
					})
				}
			}
		}()
		events = fanoutEvents
	}

	// * if skill is empty, then treat as no skill
	if data.Skill != nil && data.Skill.Content == "" {
		data.Skill = nil
	}

	scanner := data.SkillScanner
	if scanner == nil {
		scanner = agents.Scanner()
	}

	exec, err := tools.NewExecutor(data.WorkDir, session.ID, scanner)
	if err != nil {
		return fmt.Errorf("tools.NewExecutor: %w", err)
	}

	emitChangedFiles := func() {
		if files := exec.EditedFiles(); len(files) > 0 {
			events <- agentTypes.Event{Type: agentTypes.EventFileChanged, Files: files}
		}
	}

	exec.CancelExecution = execCancel

	keepPending := true
	if !session.Stateless && session.ID != "" {
		if data.PendingTask != "" {
			exec.PendingTask = data.PendingTask
			exec.SeedFiles(interactive.LoadPendingFiles(session.ID, data.PendingTask))
		} else {
			objective := data.Content
			if objective == "" {
				if s, ok := session.UserInput.Content.(string); ok {
					objective = s
				}
			}
			exec.PendingTask = interactive.CreateExecPending(session.ID, objective, data.ReplyMessageID, allowAll)
		}
		defer func() {
			if !keepPending {
				interactive.CleanupPending(session.ID, exec.PendingTask)
			}
		}()
	}

	if data.Skill != nil {
		assignBindingSkill(session, data.Skill)
	}

	cfg, _ := config.Load()
	if go_pkg_keychain.Get("GEMINI_API_KEY") == "" {
		data.ExcludeTools = append(data.ExcludeTools, "transcribe_media")
	}
	if !imageTool.Enabled() {
		data.ExcludeTools = append(data.ExcludeTools, "generate_image")
	}
	if (cfg == nil || !cfg.TelegramEnabled || go_pkg_keychain.Get("TELEGRAM_TOKEN") == "") &&
		(cfg == nil || !cfg.DiscordEnabled || go_pkg_keychain.Get("DISCORD_TOKEN") == "") {
		data.ExcludeTools = append(data.ExcludeTools,
			"list_chatbot", "send_to_chatbot")
	}
	if strings.HasPrefix(session.ID, "ln-") {
		data.ExcludeTools = append(data.ExcludeTools,
			"generate_image", "ask_user", "store_secret", "transcribe_media")
	}

	if len(data.ExcludeTools) > 0 {
		excluded := make(map[string]bool, len(data.ExcludeTools))
		var prefixes []string
		for _, name := range data.ExcludeTools {
			if prefix, ok := strings.CutSuffix(name, "*"); ok {
				prefixes = append(prefixes, prefix)
				continue
			}
			excluded[name] = true
		}
		if len(prefixes) > 0 {
			for _, t := range exec.Tools {
				if slices.ContainsFunc(prefixes, func(p string) bool {
					return strings.HasPrefix(t.Function.Name, p)
				}) {
					excluded[t.Function.Name] = true
				}
			}
		}
		exec.ExcludeTools = excluded

		filtered := exec.Tools[:0]
		for _, t := range exec.Tools {
			if !excluded[t.Function.Name] {
				filtered = append(filtered, t)
			}
		}
		exec.Tools = filtered

		for name := range excluded {
			delete(exec.StubTools, name)
		}
	}

	clientTools := make(map[string]bool, len(data.ClientTools))
	for _, t := range data.ClientTools {
		name := strings.TrimSpace(t.Function.Name)
		if name == "" {
			continue
		}
		clientTools[name] = true
		exec.Tools = append(exec.Tools, t)
	}

	limit := filesystem.MaxToolIterations
	reasoningName := data.Reasoning
	if reasoningName == "" {
		_, reasoningName = configBot.GetModel(session.ID)
	}
	reasoning, _ := provider.ParseReasoning(reasoningName)

	allAgents := make([]agentTypes.Agent, 0, 1+len(data.FallbackAgents))
	allAgents = append(allAgents, data.Agent)
	allAgents = append(allAgents, data.FallbackAgents...)
	fallbackRound := 0

	var usage provider.Usage
	alreadyCall := make(map[string]string)
	turnAllowAll := false
	emptyCount := 0
	compactFailed := false
	lastInputTokens := 0
	var shownReasoning []string
	type sendOutcome struct {
		resp        *provider.Output
		code        int
		err         error
		textEmitted bool
		reasoned    bool
	}
	sendFailCount := 0
	timeoutRetryCount := 0
	oldHistoriesCompacted := false
	firstAttempt := true

	for range limit {
		if execCtx.Err() != nil {
			events <- agentTypes.Event{Type: agentTypes.EventCanceled, Model: data.Agent.Name(), Duration: time.Since(execStart)}
			interactive.DeletePending(session.ID, exec.PendingTask)
			keepPending = false
			return execCtx.Err()
		}
		if pending := getSteer(session.ID); len(pending) > 0 {
			raw := strings.Join(pending, "\n")
			session.ToolHistories = append(session.ToolHistories, provider.Message{Role: "user", Content: formatSteerInjection(pending)})
			events <- agentTypes.Event{Type: agentTypes.EventUserInjected, Text: raw}
		}
		if firstAttempt {
			firstAttempt = false
		} else if !compactFailed && lastInputTokens >= compact.CheckThreshold(data.Agent.Name()) {
			compacted := false
			if !oldHistoriesCompacted {
				compacted = compact.ExtractOldHistories(execCtx, data.Agent, session, &usage, events)
				oldHistoriesCompacted = true
			}
			if !compacted {
				events <- agentTypes.Event{Type: agentTypes.EventCompact, Text: "tool_call"}
				compacted = compact.ToolHistory(execCtx, data.Agent, session, &usage, exec.PendingTask)
			}
			if compacted {
				lastInputTokens = 0
			} else {
				compactFailed = true
			}
		}
		assembled := compact.AssembleMessages(session, exec.PendingTask)
		sendStart := time.Now()
		sendCtx, cancelSend := context.WithTimeout(execCtx, time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
		sendAgent := data.Agent
		resultCh := make(chan sendOutcome, 1)
		sendDone := make(chan struct{})
		go func() {
			defer close(sendDone)
			r, c, textEmitted, reasoned, e := streamSend(sendCtx, sendAgent, assembled, exec.Tools, reasoning, fast.Mode(), events, &shownReasoning)
			resultCh <- sendOutcome{resp: r, code: c, err: e, textEmitted: textEmitted, reasoned: reasoned}
		}()

		stopSend := func() {
			cancelSend()
			<-sendDone
		}

		watchdog := time.NewTimer(UnresponsiveProbeInterval)
		unresponsiveFailures := 0
		var resp *provider.Output
		var sendCode int
		var err error
		var textEmitted bool
		var reasoned bool
		switched := false
	waitSend:
		for {
			select {
			case <-execCtx.Done():
				watchdog.Stop()
				stopSend()
				events <- agentTypes.Event{Type: agentTypes.EventCanceled, Model: data.Agent.Name(), Duration: time.Since(execStart)}
				interactive.DeletePending(session.ID, exec.PendingTask)
				keepPending = false
				return execCtx.Err()
			case out := <-resultCh:
				resp, sendCode, err, textEmitted, reasoned = out.resp, out.code, out.err, out.textEmitted, out.reasoned
				break waitSend
			case <-watchdog.C:
				if checkAgentResponsive(execCtx, data.Agent, HealthCheckTimeout) {
					unresponsiveFailures = 0
					watchdog.Reset(UnresponsiveProbeInterval)
					continue
				}
				unresponsiveFailures++
				if unresponsiveFailures < MaxUnresponsiveProbeFailures {
					slog.Debug("agent health probe failed, retrying",
						slog.String("session", session.ID),
						slog.String("name", data.Agent.Name()),
						slog.Int("failures", unresponsiveFailures))
					watchdog.Reset(UnresponsiveRetryInterval)
					continue
				}
				next, nextName := nextAgent(execCtx, session.ID, data.Agent.Name(), &data.FallbackAgents, allAgents, &fallbackRound, lastInputTokens)
				if next == nil {
					watchdog.Stop()
					stopSend()
					deadName := data.Agent.Name()
					slog.Error("agent unresponsive, no healthy fallback; aborting",
						slog.String("session", session.ID),
						slog.String("name", deadName))
					msg := fmt.Sprintf("upstream %s is unresponsive and no healthy fallback model is available.", deadName)
					sendText(events, msg)
					emitChangedFiles()
					events <- agentTypes.Event{
						Type:     agentTypes.EventDone,
						Model:    deadName,
						Usage:    &usage,
						Duration: time.Since(execStart),
					}
					interactive.FinalizePending(session.ID, exec.PendingTask, msg)
					keepPending = false
					return fmt.Errorf("agent %s unresponsive, no healthy fallback", deadName)
				}
				unresponsiveFailures = 0
				watchdog.Reset(UnresponsiveProbeInterval)
				slog.Debug("agent unresponsive, switching model",
					slog.String("session", session.ID),
					slog.String("from", data.Agent.Name()),
					slog.String("to", nextName))
				events <- agentTypes.Event{
					Type:  agentTypes.EventAgentResult,
					Text:  nextName,
					Model: nextName,
				}
				data.Agent = next
				switched = true
				break waitSend
			}
		}
		watchdog.Stop()
		if switched {
			stopSend()
			events <- agentTypes.Event{Type: agentTypes.EventCompact, Text: "tool_call"}
			if !compact.RawToolFallback(session, exec.PendingTask) {
				session.ToolHistories = nil
			}
			alreadyCall = make(map[string]string)
			sendFailCount = 0
			timeoutRetryCount = 0
			emptyCount = 0
			compactFailed = false
			continue
		}
		sendElapsed := time.Since(sendStart).Round(time.Second)
		sendCtxErr := sendCtx.Err()
		stopSend()
		if err != nil {
			if execCtx.Err() != nil {
				events <- agentTypes.Event{Type: agentTypes.EventCanceled, Model: data.Agent.Name(), Duration: time.Since(execStart)}
				interactive.DeletePending(session.ID, exec.PendingTask)
				keepPending = false
				return execCtx.Err()
			}
			isTimeout := isSendTimeoutError(err, sendCtxErr)
			modelName := data.Agent.Name()

			if reason := cooldown.Reason(err, sendCode); reason != "" {
				cooldown.Register(modelName)
				slog.Debug("data.Agent.Send "+reason+", model cooldown registered",
					slog.String("session", session.ID),
					slog.String("name", modelName))
			}

			if compact.IsContextLengthError(err) {
				if len(session.OldHistories) == 0 && len(session.ToolHistories) == 0 {
					slog.Error("data.Agent.Send context length exceeded, nothing left to trim",
						slog.String("session", session.ID),
						slog.String("error", err.Error()),
						slog.Int("attempts", sendFailCount))
					msg := fmt.Sprintf("upstream %s context exceeded and nothing left to trim. Start a new session or switch to a larger-context model.", modelName)
					sendText(events, msg)
					emitChangedFiles()
					events <- agentTypes.Event{
						Type:     agentTypes.EventDone,
						Model:    modelName,
						Usage:    &usage,
						Duration: time.Since(execStart),
					}
					interactive.FinalizePending(session.ID, exec.PendingTask, msg)
					keepPending = false
					return fmt.Errorf("data.Agent.Send context exceeded, nothing left to trim: %w", err)
				}
				sendFailCount++
				compact.TrimFallback(&session.OldHistories, &session.ToolHistories)
				slog.Warn("data.Agent.Send context length exceeded, trimming oldest exchange",
					slog.String("session", session.ID),
					slog.Int("attempts", sendFailCount))
				continue
			}

			slog.Debug("data.Agent.Send",
				slog.String("session", session.ID),
				slog.String("error", err.Error()),
				slog.Bool("timeout", isTimeout))

			if isTimeout && timeoutRetryCount < MaxSendTimeoutRetries-1 {
				timeoutRetryCount++
				slog.Debug("data.Agent.Send timed out, retrying same model",
					slog.String("session", session.ID),
					slog.String("name", modelName),
					slog.Int("attempt", timeoutRetryCount+1))
				select {
				case <-execCtx.Done():
					events <- agentTypes.Event{Type: agentTypes.EventCanceled, Model: data.Agent.Name(), Duration: time.Since(execStart)}
					interactive.DeletePending(session.ID, exec.PendingTask)
					keepPending = false
					return execCtx.Err()
				case <-time.After(SendTimeoutRetryInterval):
				}
				continue
			}

			next, nextName := nextAgent(execCtx, session.ID, modelName, &data.FallbackAgents, allAgents, &fallbackRound, lastInputTokens)
			if next != nil {
				slog.Debug("data.Agent.Send failed, switching model",
					slog.String("session", session.ID),
					slog.String("from", modelName),
					slog.String("to", nextName))
				events <- agentTypes.Event{
					Type:  agentTypes.EventAgentResult,
					Text:  nextName,
					Model: nextName,
				}
				data.Agent = next
				events <- agentTypes.Event{Type: agentTypes.EventCompact, Text: "tool_call"}
				if !compact.RawToolFallback(session, exec.PendingTask) {
					session.ToolHistories = nil
				}
				alreadyCall = make(map[string]string)
				sendFailCount = 0
				timeoutRetryCount = 0
				emptyCount = 0
				compactFailed = false
				continue
			}

			var userMsg string
			if isTimeout {
				userMsg = fmt.Sprintf("upstream %s timed out (%s) and no healthy fallback model is available.", modelName, sendElapsed)
			} else {
				userMsg = fmt.Sprintf("upstream %s failed and no healthy fallback model is available: %s", modelName, err.Error())
			}
			slog.Error("data.Agent.Send failed, no fallback",
				slog.String("session", session.ID),
				slog.String("error", err.Error()))
			sendText(events, userMsg)
			emitChangedFiles()
			events <- agentTypes.Event{
				Type:     agentTypes.EventDone,
				Model:    modelName,
				Usage:    &usage,
				Duration: time.Since(execStart),
			}
			interactive.FinalizePending(session.ID, exec.PendingTask, userMsg)
			keepPending = false
			return fmt.Errorf("data.Agent.Send failed: %w", err)
		}
		cooldown.Clear(data.Agent.Name())
		sendFailCount = 0
		timeoutRetryCount = 0

		usage.Input += resp.Usage.Input
		usage.Output += resp.Usage.Output
		usage.CacheCreate += resp.Usage.CacheCreate
		usage.CacheRead += resp.Usage.CacheRead
		lastInputTokens = resp.Usage.Input + resp.Usage.CacheRead

		prov, model, _ := strings.Cut(data.Agent.Name(), "@")
		usagelog.Append(session.ID, prov, model, resp.Usage)

		usageSnapshot := usage
		events <- agentTypes.Event{Type: agentTypes.EventUsageUpdate, Usage: &usageSnapshot}

		if len(resp.Choices) == 0 {
			if emptyRetryExhausted(&emptyCount, events, session.ID, exec.PendingTask, data.Agent.Name(), "no choices", &usage, execStart) {
				keepPending = false
				return nil
			}
			continue
		}

		choice := resp.Choices[0]
		if choice.Message.ReasoningContent == "" {
			if s, ok := choice.Message.Content.(string); ok {
				if think, rest := splitThinkTag(s); think != "" {
					choice.Message.ReasoningContent = think
					choice.Message.Content = rest
				}
			}
		}

		if reasoned {
			markReasoningShown(choice.Message.ReasoningContent, &shownReasoning)
		} else {
			emitReasoning(events, choice.Message.ReasoningContent, &shownReasoning)
		}
		choice.Message.ReasoningContent = ""

		if len(choice.Message.ToolCalls) > 0 {
			emptyCount = 0
			if text, ok := choice.Message.Content.(string); ok {
				if stripped := StripModelResponse(text); stripped != "" && !isGuardrailRefusal(stripped) {
					if textEmitted {
						events <- agentTypes.Event{Type: agentTypes.EventTextDone}
					} else {
						sendText(events, stripped)
					}
				}
			}
			if len(clientTools) > 0 {
				var handoff []provider.ToolCall
				for _, call := range choice.Message.ToolCalls {
					if clientTools[strings.TrimSpace(call.Function.Name)] {
						handoff = append(handoff, call)
					}
				}
				if len(handoff) > 0 {
					events <- agentTypes.Event{Type: agentTypes.EventClientToolCall, ClientToolCalls: handoff}
					emitChangedFiles()
					events <- agentTypes.Event{
						Type:     agentTypes.EventDone,
						Model:    data.Agent.Name(),
						Usage:    &usage,
						Duration: time.Since(execStart),
					}
					keepPending = false
					return nil
				}
			}

			session, alreadyCall, err = toolCall(execCtx, exec, choice, session, events, allowAll, alreadyCall, &turnAllowAll)
			if err != nil {
				if errors.Is(err, ErrAskUserInterrupted) {
					return nil
				}
				return err
			}
			continue
		}

		switch value := choice.Message.Content.(type) {
		case string:
			str := value
			if str == "" {
				if emptyRetryExhausted(&emptyCount, events, session.ID, exec.PendingTask, data.Agent.Name(), "empty content", &usage, execStart) {
					keepPending = false
					return nil
				}
				continue
			}

			stripped := StripModelResponse(str)
			if stripped == "" {
				if emptyRetryExhausted(&emptyCount, events, session.ID, exec.PendingTask, data.Agent.Name(), "content stripped to empty", &usage, execStart) {
					keepPending = false
					return nil
				}
				continue
			}
			emptyCount = 0

			if isGuardrailRefusal(stripped) {
				sendText(events, configs.PoisonRefusal)
				emitChangedFiles()
				events <- agentTypes.Event{Type: agentTypes.EventDone, Model: data.Agent.Name(), Usage: &usage, Duration: time.Since(execStart)}
				interactive.FinalizePending(session.ID, exec.PendingTask, configs.PoisonRefusal)
				keepPending = false
				return nil
			}

			responseText := stripped
			if textEmitted {
				events <- agentTypes.Event{Type: agentTypes.EventTextDone}
			} else {
				sendText(events, responseText)
			}

			choice.Message.Content = sessionHistory.WithPrefix(
				sessionHistory.Record{SendAt: time.Now().UnixNano()}.Prefix(),
				stripped,
			)
			session.ToolHistories = append(session.ToolHistories, choice.Message)

			if err := saveNewHistory(execCtx, choice, session); err != nil {
				slog.Warn("writeHistory",
					slog.String("session", session.ID),
					slog.String("error", err.Error()))
			}

			interactive.FinalizePending(session.ID, exec.PendingTask, responseText)

			if pending := getSteer(session.ID); len(pending) > 0 {
				session.ToolHistories = append(session.ToolHistories, provider.Message{Role: "user", Content: formatSteerInjection(pending)})
				events <- agentTypes.Event{Type: agentTypes.EventUserInjected, Text: strings.Join(pending, "\n")}
				continue
			}

		case nil:
			if emptyRetryExhausted(&emptyCount, events, session.ID, exec.PendingTask, data.Agent.Name(), "nil content", &usage, execStart) {
				keepPending = false
				return nil
			}
			continue

		default:
			return fmt.Errorf("unexpected content type: %T", choice.Message.Content)
		}

		emitChangedFiles()
		events <- agentTypes.Event{Type: agentTypes.EventDone, Model: data.Agent.Name(), Usage: &usage, Duration: time.Since(execStart)}

		keepPending = false
		return nil
	}

	assembled := compact.AssembleMessages(session, exec.PendingTask)
	summaryMessages := append(assembled, provider.Message{
		Role:    "user",
		Content: "請根據以上工具查詢結果，整理並總結回答原始問題。",
	})
	resp, _, err := data.Agent.Send(execCtx, summaryMessages, nil, reasoning, fast.Mode())
	if err == nil {
		cooldown.Clear(data.Agent.Name())
	}
	if err == nil && len(resp.Choices) > 0 {
		usage.Input += resp.Usage.Input
		usage.Output += resp.Usage.Output
		usage.CacheCreate += resp.Usage.CacheCreate
		usage.CacheRead += resp.Usage.CacheRead

		prov, model, _ := strings.Cut(data.Agent.Name(), "@")
		usagelog.Append(session.ID, prov, model, resp.Usage)

		emitReasoning(events, resp.Choices[0].Message.ReasoningContent, &shownReasoning)
		if text, ok := resp.Choices[0].Message.Content.(string); ok && text != "" {
			summaryStripped := StripModelResponse(text)
			if isGuardrailRefusal(summaryStripped) {
				sendText(events, configs.PoisonRefusal)
				emitChangedFiles()
				events <- agentTypes.Event{Type: agentTypes.EventDone, Model: data.Agent.Name(), Usage: &usage, Duration: time.Since(execStart)}
				interactive.FinalizePending(session.ID, exec.PendingTask, configs.PoisonRefusal)
				keepPending = false
				return nil
			}
			sendText(events, summaryStripped)
			emitChangedFiles()
			events <- agentTypes.Event{Type: agentTypes.EventDone, Model: data.Agent.Name(), Usage: &usage, Duration: time.Since(execStart)}
			interactive.FinalizePending(session.ID, exec.PendingTask, summaryStripped)
			keepPending = false
			return nil
		}
	}

	slog.Error("tool loop exhausted without a usable final answer",
		slog.String("session", session.ID),
		slog.String("name", data.Agent.Name()))
	sendEmptyData(events, session.ID, exec.PendingTask, data.Agent.Name(), &usage, execStart)
	keepPending = false
	return nil
}
