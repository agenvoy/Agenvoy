package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	allowTool "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/tool"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/memory"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	"github.com/pardnchiu/agenvoy/internal/tools"
	"github.com/pardnchiu/agenvoy/internal/tools/file"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	"github.com/pardnchiu/agenvoy/internal/tools/toolcache"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func askUserInBackground(sessionID, taskHash, rawArgs string, toolResults []interactive.ToolResult) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("askUserInBackground panic recovered",
				slog.String("session", sessionID),
				slog.Any("panic", r))
		}
	}()
	var params struct {
		Questions []runtime.Question `json:"questions"`
		State     struct {
			Objective string   `json:"objective"`
			Completed []string `json:"completed"`
			NextSteps []string `json:"next_steps"`
		} `json:"state"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &params); err != nil {
		slog.Warn("json Unmarshal",
			slog.String("error", err.Error()))
		return
	}
	if len(params.Questions) == 0 {
		slog.Warn("ask user no questions")
		return
	}

	hash := interactive.SaveAndEnqueueAskUser(sessionID, params.Questions, params.State.Objective, params.State.Completed, params.State.NextSteps, toolResults, taskHash)
	pubsub.Pub(sessionID, agentTypes.Event{Type: agentTypes.EventPending, Text: hash})
}

const confirmTimeout = 5 * time.Minute

func isTUISession(sessionID string) bool {
	return strings.HasPrefix(strings.TrimSpace(sessionID), "cli-")
}

var ErrAskUserInterrupted = errors.New("ask user interrupted")

func toolResults(session *agentTypes.AgentSession) []interactive.ToolResult {
	nameByID := make(map[string]string)
	for _, msg := range session.ToolHistories {
		for _, tc := range msg.ToolCalls {
			nameByID[tc.ID] = tc.Function.Name
		}
	}

	var results []interactive.ToolResult
	for _, msg := range session.Tools {
		content, _ := msg.Content.(string)
		results = append(results, interactive.ToolResult{
			Name:   nameByID[msg.ToolCallID],
			ID:     msg.ToolCallID,
			Result: content,
		})
	}
	return results
}

const (
	slotReady          = 0
	slotCached         = 1
	slotSkipped        = 2
	slotStubActivated  = 3
	slotValidateFailed = 4
	slotDispatched     = 5
)

type toolSlot struct {
	idx  int
	id   string
	name string
	args string
	hash string

	state     int
	preMsg    string
	imageURLs []string

	result     string
	execErr    string
	execErrVal error
}

func toolNeedsConfirmation(exec *toolTypes.Executor, toolName, toolArgs string, turnAllowAll bool) bool {
	if toolName == "read_files" && isSensitiveReadFile(toolArgs) {
		return true
	}
	if turnAllowAll {
		return false
	}
	if isDestructiveMode(toolArgs) {
		return true
	}
	if toolRegister.IsReadOnly(toolName) {
		return false
	}
	if isReadOnlyMode(toolArgs) {
		return false
	}
	if toolName == "http_request" && isGet(toolArgs) {
		return false
	}
	if toolName == "run_command" && isReadOnlyRunCommand(toolArgs) {
		return false
	}
	return !allowTool.Match(allowTool.List(exec.WorkDir), toolName, toolArgs)
}

var readOnlyModes = map[string]bool{
	"list":   true,
	"read":   true,
	"search": true,
}

var destructiveModes = map[string]bool{
	"remove":  true,
	"restore": true,
}

func isDestructiveMode(toolArgs string) bool {
	var p struct {
		Mode string `json:"mode"`
	}
	if json.Unmarshal([]byte(toolArgs), &p) != nil {
		return false
	}
	return destructiveModes[strings.TrimSpace(p.Mode)]
}

func isReadOnlyMode(toolArgs string) bool {
	var p struct {
		Mode string `json:"mode"`
	}
	if json.Unmarshal([]byte(toolArgs), &p) != nil {
		return false
	}
	return readOnlyModes[strings.TrimSpace(p.Mode)]
}

func hasDangerousGitFlag(args []string) bool {
	for _, a := range args {
		switch {
		case a == "-o", a == "--output", a == "--output-directory":
			return true
		case strings.HasPrefix(a, "--output=") || strings.HasPrefix(a, "--output-directory="):
			return true
		case strings.HasPrefix(a, "-o") && a != "-o":
			return true
		}
	}
	return false
}

func stripSafeGitGlobalFlags(argv []string) []string {
	i := 1
	for i < len(argv) && strings.HasPrefix(argv[i], "-") {
		if argv[i] == "-c" || strings.HasPrefix(argv[i], "-c=") {
			break
		}
		if !strings.Contains(argv[i], "=") && i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
			i += 2
		} else {
			i++
		}
	}
	return append([]string{argv[0]}, argv[i:]...)
}

func isReadOnlyRunCommand(toolArgs string) bool {
	var p struct {
		Argv []string `json:"argv"`
	}
	if json.Unmarshal([]byte(toolArgs), &p) != nil || len(p.Argv) == 0 {
		return false
	}
	argv := p.Argv
	bin := filepath.Base(argv[0])
	if bin == "git" {
		argv = stripSafeGitGlobalFlags(argv)
	}

	matched := slices.Contains(filesystem.ReadOnlyCommand, bin)
	if !matched && len(argv) > 1 {
		matched = slices.Contains(filesystem.ReadOnlyCommand, bin+" "+argv[1])
	}
	if !matched {
		return false
	}
	if bin == "git" {
		return !hasDangerousGitFlag(argv[2:])
	}
	return true
}

func invalidateReadFileCache(alreadyCall map[string]string, writeArgsJSON string) {
	var p struct {
		Path string `json:"path"`
	}
	if json.Unmarshal([]byte(writeArgsJSON), &p) != nil || p.Path == "" {
		return
	}
	for key := range alreadyCall {
		if strings.HasPrefix(key, "read_files|") && strings.Contains(key, p.Path) {
			delete(alreadyCall, key)
		}
	}
}

var isWriteLikeTool = map[string]bool{
	"edit_file":  true,
	"edit_skill": true,
	"edit_tool":  true,
}

func truncateWriteArgs(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) != nil {
		return argsJSON
	}
	const omitted = "[ARGUMENT ELIDED FROM HISTORY TO SAVE CONTEXT — NOT THE FILE'S CONTENT. The full text was sent and written to disk successfully. Do NOT re-write this file to restore it.]"
	const maxKeptAnchor = 2 << 10

	dics := []map[string]any{m}
	if targets, ok := m["targets"].([]any); ok {
		for _, t := range targets {
			if tm, ok := t.(map[string]any); ok {
				dics = append(dics, tm)
			}
		}
	}
	for _, dic := range dics {
		for _, field := range []string{"content", "new_string"} {
			if _, ok := dic[field]; ok {
				dic[field] = omitted
			}
		}
		if str, ok := dic["old_string"].(string); ok && len(str) > maxKeptAnchor {
			dic["old_string"] = omitted
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return argsJSON
	}
	return string(out)
}

var checkpointClearableTool = map[string]bool{
	"find_files":  true,
	"run_command": true,
}

func hasCompletedTodo(argsJSON string) bool {
	var p struct {
		Todos []struct {
			Status string `json:"status"`
		} `json:"todos"`
	}
	if json.Unmarshal([]byte(argsJSON), &p) != nil {
		return false
	}
	for _, t := range p.Todos {
		if t.Status == agentTypes.TodoCompleted {
			return true
		}
	}
	return false
}

func clearCheckpointedToolResults(sessionData *agentTypes.AgentSession) {
	start := sessionData.ToolCheckpoint
	if start < 0 || start >= len(sessionData.ToolHistories) {
		sessionData.ToolCheckpoint = len(sessionData.ToolHistories)
		return
	}

	segment := sessionData.ToolHistories[start:]
	nameByID := make(map[string]string, len(segment))
	for _, msg := range segment {
		for _, tc := range msg.ToolCalls {
			nameByID[tc.ID] = tc.Function.Name
		}
	}

	const cleared = "[cleared after step completed — already acted on]"
	for i := range segment {
		msg := &segment[i]
		if msg.Role != "tool" || msg.ToolCallID == "" {
			continue
		}
		if !checkpointClearableTool[nameByID[msg.ToolCallID]] {
			continue
		}
		if content, ok := msg.Content.(string); ok && content != cleared {
			msg.Content = cleared
		}
	}

	sessionData.ToolCheckpoint = len(sessionData.ToolHistories)
}

func restrictedList(paths, commands []string) []string {
	out := make([]string, 0, len(paths)+len(commands))
	out = append(out, paths...)
	for _, one := range commands {
		out = append(out, "command: "+one)
	}
	return out
}

func isSensitiveReadFile(argsJSON string) bool {
	var p struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if json.Unmarshal([]byte(argsJSON), &p) != nil {
		return false
	}
	for _, f := range p.Files {
		if f.Path != "" && file.IsSensitivePath(f.Path) {
			return true
		}
	}
	return false
}

func isGet(argsJSON string) bool {
	var p struct {
		Method string `json:"method"`
	}
	if json.Unmarshal([]byte(argsJSON), &p) != nil {
		return false
	}
	return p.Method == "" || strings.EqualFold(p.Method, "GET")
}

func toolCall(ctx context.Context, exec *toolTypes.Executor, choice provider.OutputChoices, sessionData *agentTypes.AgentSession, events chan<- agentTypes.Event, allowAll bool, alreadyCall map[string]string, turnAllowAll *bool) (*agentTypes.AgentSession, map[string]string, error) {
	sessionData.ToolHistories = append(sessionData.ToolHistories, choice.Message)

	calls := choice.Message.ToolCalls
	slots := make([]toolSlot, len(calls))
	activatedInBatch := make(map[string]bool)

	for i, tool := range calls {
		toolID := strings.TrimSpace(tool.ID)
		toolArg := strings.TrimSpace(tool.Function.Arguments)
		toolName := strings.TrimSpace(tool.Function.Name)
		if idx := strings.Index(toolName, "<|"); idx != -1 {
			toolName = toolName[:idx]
		}
		hashArg := toolArg
		var argMap map[string]any
		if json.Unmarshal([]byte(toolArg), &argMap) == nil {
			if normalized, err := json.Marshal(argMap); err == nil {
				hashArg = string(normalized)
			}
		}
		hash := fmt.Sprintf("%v|%v", toolName, hashArg)

		slots[i] = toolSlot{
			idx:   i,
			id:    toolID,
			name:  toolName,
			args:  toolArg,
			hash:  hash,
			state: slotReady,
		}

		interactive.RecordToolAttempt(exec.SessionID, exec.PendingTask, interactive.ToolAttempt{
			Name: toolName,
			ID:   toolID,
			Args: toolArg,
		})

		if cached, ok := alreadyCall[hash]; ok && cached != "" {
			cachedContent := strings.TrimSpace(cached)
			if images, rest := splitImageResult(cached); len(images) > 0 {
				cachedContent = imageLoadedMessage(toolName, len(images), rest)
				slots[i].imageURLs = images
			}
			slots[i].state = slotCached
			slots[i].preMsg = cachedContent
			continue
		}

		if exec.StubTools[toolName] || activatedInBatch[toolName] {
			if exec.StubTools[toolName] {
				activateArgs, _ := json.Marshal(map[string]any{"mode": "search", "query": "select:" + toolName})
				if _, err := toolRegister.Dispatch(ctx, exec, "find_tools", activateArgs); err != nil {
					slog.Warn("stub tool activation failed",
						slog.String("name", toolName),
						slog.String("error", err.Error()))
				}
				delete(exec.StubTools, toolName)
			}
			activatedInBatch[toolName] = true
			slots[i].state = slotStubActivated
			slots[i].preMsg = fmt.Sprintf("[%s] tool schema just loaded. Re-invoke %s with the correct arguments — the previous call was made against a stub with empty params.", toolName, toolName)
			continue
		}

		restrictedPaths := boundary.Restricted(exec.SessionID, exec.WorkDir, toolName, toolArg)
		restrictedCmds := tools.RestrictedCommands(exec.AllowedCommand, toolName, toolArg)
		restricted := len(restrictedPaths) > 0 || len(restrictedCmds) > 0

		if !allowAll && (restricted || toolNeedsConfirmation(exec, toolName, toolArg, *turnAllowAll)) {
			proceed := true
			approved := false
			verified := false
			reason := ""
			if runtime.HasListener(sessionData.ID) {
				askCtx := ctx
				cancelAsk := func() {}
				if !isTUISession(sessionData.ID) {
					askCtx, cancelAsk = context.WithTimeout(ctx, confirmTimeout)
				}
				reply, err := runtime.Ask(askCtx, runtime.Request{
					Kind:       runtime.KindToolConfirm,
					SessionID:  sessionData.ID,
					ToolName:   toolName,
					ToolArgs:   toolArg,
					Restricted: restrictedList(restrictedPaths, restrictedCmds),
				})
				cancelAsk()
				if !isTUISession(sessionData.ID) && errors.Is(err, context.DeadlineExceeded) {
					events <- agentTypes.Event{
						Type:     agentTypes.EventToolSkipped,
						ToolName: toolName,
						ToolArgs: toolArg,
						ToolID:   toolID,
						Text:     "no answer within " + confirmTimeout.String() + "; task kept as pending",
					}
					if exec.CancelExecution != nil {
						exec.CancelExecution()
					}
					return sessionData, alreadyCall, fmt.Errorf("tool confirmation timed out after %s; resume from pending to continue", confirmTimeout)
				}
				if err != nil {
					proceed = false
				} else {
					proceed = reply.Approve
					approved = reply.Approve
					verified = reply.Verified
					reason = reply.Reason
					if reply.Approve && reply.Remember {
						if err = allowTool.Append(exec.WorkDir, toolName, toolArg); err != nil {
							slog.Warn("appendAllowListRule",
								slog.String("session", sessionData.ID),
								slog.String("error", err.Error()))
						}
					}
					if reply.Approve && reply.AllowTurn {
						*turnAllowAll = true
					}
				}
			}
			if approved && restricted && !verified {
				proceed = false
				approved = false
				reason = fmt.Sprintf("this needs system password verification, which this channel cannot collect: %s", strings.Join(restrictedList(restrictedPaths, restrictedCmds), ", "))
			}
			if !proceed {
				message := "Skipped by user"
				if reason != "" {
					message = fmt.Sprintf("Skipped by user. Reason: %s", reason)
				}
				events <- agentTypes.Event{
					Type:     agentTypes.EventToolSkipped,
					ToolName: toolName,
					ToolArgs: toolArg,
					ToolID:   toolID,
					Text:     reason,
				}
				slots[i].state = slotSkipped
				slots[i].preMsg = message
				continue
			}
			if approved {
				boundary.Grant(exec.SessionID, restrictedPaths...)
				tools.GrantCommands(exec.SessionID, restrictedCmds)
			}
		}

		if earlyErr := validateToolArgs(exec, toolName, toolArg); earlyErr != "" {
			events <- agentTypes.Event{
				Type:     agentTypes.EventToolCall,
				ToolName: toolName,
				ToolArgs: toolArg,
				ToolID:   toolID,
			}
			content := fmt.Sprintf("tool=%s failed: %s", toolName, earlyErr)
			slots[i].state = slotValidateFailed
			slots[i].preMsg = content
			continue
		}
	}

	for i := range slots {
		slot := &slots[i]
		if slot.state == slotReady && slot.name == "ask_user" {
			for j := range slots {
				cs := &slots[j]
				if cs.state == slotReady || cs.name == "ask_user" {
					continue
				}
				content := cs.preMsg
				msg := provider.Message{
					Role:       "tool",
					Content:    content,
					ToolCallID: cs.id,
				}
				switch cs.state {
				case slotCached:
					for _, url := range cs.imageURLs {
						injectImageToUserInput(sessionData, url)
					}
					sessionData.ToolHistories = append(sessionData.ToolHistories, msg)
				default:
					sessionData.Tools = append(sessionData.Tools, msg)
					sessionData.ToolHistories = append(sessionData.ToolHistories, msg)
				}
			}

			toolResults := toolResults(sessionData)

			go askUserInBackground(sessionData.ID, exec.PendingTask, slot.args, toolResults)
			if exec.CancelExecution != nil {
				exec.CancelExecution()
			}
			return sessionData, alreadyCall, ErrAskUserInterrupted
		}
	}

	var wg sync.WaitGroup
	for i := range slots {
		s := &slots[i]
		if s.state != slotReady {
			continue
		}
		if toolRegister.IsBackground(s.name) {
			go runToolExec(ctx, exec, s, events)
			s.result = "ok"
			s.state = slotDispatched
			continue
		}
		if toolRegister.IsConcurrent(s.name) {
			wg.Add(1)
			go func(s *toolSlot) {
				defer wg.Done()
				runToolExec(ctx, exec, s, events)
			}(s)
			s.state = slotDispatched
			continue
		}
	}
	for i := range slots {
		s := &slots[i]
		if s.state != slotReady {
			continue
		}
		runToolExec(ctx, exec, s, events)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return sessionData, alreadyCall, err
	}

	todoCheckpointHit := false

	for i := range slots {
		s := &slots[i]
		switch s.state {
		case slotCached:
			for _, url := range s.imageURLs {
				injectImageToUserInput(sessionData, url)
			}
			sessionData.ToolHistories = append(sessionData.ToolHistories, provider.Message{
				Role:       "tool",
				Content:    s.preMsg,
				ToolCallID: s.id,
			})
			continue
		case slotSkipped, slotStubActivated, slotValidateFailed:
			msg := provider.Message{
				Role:       "tool",
				Content:    s.preMsg,
				ToolCallID: s.id,
			}
			sessionData.Tools = append(sessionData.Tools, msg)
			sessionData.ToolHistories = append(sessionData.ToolHistories, msg)
			continue
		}

		result := s.result
		historyResult := ""
		if s.execErr != "" {
			hint := memory.Search(ctx, s.name, s.execErr, 3)
			if hint != "" {
				result = fmt.Sprintf("tool=%s failed: %s\nrelated_errors: %s", s.name, s.execErr, hint)
			} else {
				result = fmt.Sprintf("tool=%s failed: %s", s.name, s.execErr)
			}
		} else if result == "" || result == "no data" {
			if hint := memory.Search(ctx, s.name, "no data", 3); hint != "" {
				result = hint
			} else {
				result = "no data"
			}
		}

		if s.name == "edit_file" && s.execErr == "" {
			invalidateReadFileCache(alreadyCall, s.args)
		}
		if s.name == "write_todo" && s.execErr == "" && hasCompletedTodo(s.args) {
			todoCheckpointHit = true
		}
		if isWriteLikeTool[s.name] && s.execErr == "" {
			calls[i].Function.Arguments = truncateWriteArgs(calls[i].Function.Arguments)
		}
		alreadyCall[s.hash] = result
		if images, _ := splitImageResult(result); s.execErr == "" && len(images) == 0 && toolcache.IsCacheable(s.name) {
			toolcache.Store(exec.SessionID, s.id, s.name, s.args, result)
		}

		events <- agentTypes.Event{
			Type:     agentTypes.EventToolResult,
			ToolName: s.name,
			ToolArgs: s.args,
			ToolID:   s.id,
			Result:   result,
		}

		toolMsgContent := strings.TrimSpace(fmt.Sprintf("[%s] %s", s.name, result))
		if images, rest := splitImageResult(result); len(images) > 0 {
			toolMsgContent = imageLoadedMessage(s.name, len(images), rest)
			for _, url := range images {
				injectImageToUserInput(sessionData, url)
			}
		}
		toolMsg := provider.Message{
			Role:       "tool",
			Content:    toolMsgContent,
			ToolCallID: s.id,
		}
		sessionData.Tools = append(sessionData.Tools, toolMsg)
		if historyResult != "" {
			sessionData.ToolHistories = append(sessionData.ToolHistories, provider.Message{
				Role:       "tool",
				Content:    historyResult,
				ToolCallID: s.id,
			})
		} else {
			sessionData.ToolHistories = append(sessionData.ToolHistories, toolMsg)
		}
	}

	if todoCheckpointHit {
		clearCheckpointedToolResults(sessionData)
	}

	return sessionData, alreadyCall, nil
}

func failToolEvent(exec *toolTypes.Executor, s *toolSlot, events chan<- agentTypes.Event, err error) {
	s.execErr = err.Error()
	s.execErrVal = err
	go interactive.AppendToolResult(exec.SessionID, exec.PendingTask, interactive.ToolResult{
		Name:   s.name,
		ID:     s.id,
		Result: "error: " + err.Error(),
	})
	events <- agentTypes.Event{
		Type:     agentTypes.EventToolCallEnd,
		ToolName: s.name,
		ToolID:   s.id,
	}
}

func runToolExec(ctx context.Context, exec *toolTypes.Executor, s *toolSlot, events chan<- agentTypes.Event) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		slog.Error("runToolExec panic recovered",
			slog.String("tool", s.name),
			slog.Any("panic", r))
		failToolEvent(exec, s, events, fmt.Errorf("tool %s panicked: %v", s.name, r))
	}()
	events <- agentTypes.Event{
		Type:     agentTypes.EventToolCall,
		ToolName: s.name,
		ToolArgs: s.args,
		ToolID:   s.id,
	}
	events <- agentTypes.Event{
		Type:     agentTypes.EventToolCallStart,
		ToolName: s.name,
		ToolID:   s.id,
	}
	result, err := tools.Execute(ctx, exec, s.name, json.RawMessage(s.args))
	if err != nil {
		failToolEvent(exec, s, events, err)
		return
	}

	if result != "" {
		events <- agentTypes.Event{
			Type:     agentTypes.EventToolCallText,
			ToolName: s.name,
			ToolID:   s.id,
			Text:     result,
		}
	}
	s.result = result
	go interactive.AppendToolResult(exec.SessionID, exec.PendingTask, interactive.ToolResult{
		Name:   s.name,
		ID:     s.id,
		Result: result,
	})
	if s.name == "write_todo" {
		if todos := interactive.LoadTodos(exec.SessionID, exec.PendingTask); len(todos) > 0 {
			events <- agentTypes.Event{
				Type:  agentTypes.EventTodoUpdate,
				Todos: todos,
			}
		}
	}
	events <- agentTypes.Event{
		Type:     agentTypes.EventToolCallEnd,
		ToolName: s.name,
		ToolID:   s.id,
	}
}

func validateToolArgs(exec *toolTypes.Executor, toolName, args string) string {
	if exec == nil {
		return ""
	}
	required := requiredFields(exec, toolName)
	if len(required) == 0 {
		return ""
	}

	args = strings.TrimSpace(args)
	var parsed map[string]any
	if args != "" && args != "null" {
		if err := json.Unmarshal([]byte(args), &parsed); err != nil {
			return fmt.Sprintf("invalid JSON for %s: %s. Re-send arguments as a JSON object with required fields: %s",
				toolName, err.Error(), strings.Join(required, ", "))
		}
	}

	var missing []string
	for _, f := range required {
		v, ok := parsed[f]
		if !ok {
			missing = append(missing, f)
			continue
		}
		if s, isStr := v.(string); isStr && strings.TrimSpace(s) == "" {
			missing = append(missing, f)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("missing required field(s) %s for %s. All required fields: %s",
		strings.Join(missing, ", "), toolName, strings.Join(required, ", "))
}

func requiredFields(exec *toolTypes.Executor, toolName string) []string {
	lookup := func(list []provider.Tool) []string {
		for _, t := range list {
			if t.Function.Name != toolName {
				continue
			}
			if len(t.Function.Parameters) == 0 {
				return nil
			}
			var schema struct {
				Required []string `json:"required"`
			}
			if err := json.Unmarshal(t.Function.Parameters, &schema); err != nil {
				return nil
			}
			return schema.Required
		}
		return nil
	}
	if r := lookup(exec.AllTools); len(r) > 0 {
		return r
	}
	return lookup(exec.Tools)
}

func splitImageResult(result string) (images []string, rest string) {
	trimmed := strings.TrimSpace(result)
	if strings.HasPrefix(trimmed, "data:image/") {
		return []string{trimmed}, ""
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, result
	}

	var dic map[string]string
	if json.Unmarshal([]byte(trimmed), &dic) != nil {
		return nil, result
	}

	kept := make(map[string]string, len(dic))
	for _, path := range slices.Sorted(maps.Keys(dic)) {
		if strings.HasPrefix(dic[path], "data:image/") {
			images = append(images, dic[path])
			continue
		}
		kept[path] = dic[path]
	}
	if len(images) == 0 {
		return nil, result
	}
	if len(kept) == 0 {
		return images, ""
	}
	raw, err := json.Marshal(kept)
	if err != nil {
		return images, result
	}
	return images, string(raw)
}

func imageLoadedMessage(toolName string, count int, rest string) string {
	msg := fmt.Sprintf("[%s] %d image(s) loaded", toolName, count)
	if rest != "" {
		msg += " " + rest
	}
	return msg
}

func injectImageToUserInput(session *agentTypes.AgentSession, dataURL string) {
	part := provider.ContentPart{
		Type:     "image_url",
		ImageURL: &provider.ImageURL{URL: dataURL, Detail: "auto"},
	}
	switch v := session.UserInput.Content.(type) {
	case []provider.ContentPart:
		session.UserInput.Content = append(v, part)
	case string:
		session.UserInput.Content = []provider.ContentPart{
			{Type: "text", Text: v},
			part,
		}
	}
}
