package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	allowTool "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/tool"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/memory"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/sudo"
	"github.com/pardnchiu/agenvoy/internal/tools"
	"github.com/pardnchiu/agenvoy/internal/tools/file"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	"github.com/pardnchiu/agenvoy/internal/tools/toolcache"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func askUserInBackground(sessionID, taskHash, rawArgs string, toolResults []interactive.ToolResult) {
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

	interactive.SaveAndEnqueueAskUser(sessionID, params.Questions, params.State.Objective, params.State.Completed, params.State.NextSteps, toolResults, taskHash)
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

	state    int
	preMsg   string
	isImage  bool
	imageURL string

	result     string
	execErr    string
	execErrVal error
}

func toolNeedsConfirmation(exec *toolTypes.Executor, toolName, toolArgs string, turnAllowAll bool) bool {
	if toolName == "read_files" && isSensitiveReadFile(toolArgs) {
		return true
	}
	if turnAllowAll || toolRegister.IsReadOnly(toolName) {
		return false
	}
	if toolName == "send_http_request" && isGet(toolArgs) {
		return false
	}
	if toolName == "run_command" && isReadOnlyRunCommand(toolArgs) {
		return false
	}
	return !allowTool.Match(allowTool.List(exec.WorkDir), toolName, toolArgs)
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
	"write_file":  true,
	"patch_file":  true,
	"write_skill": true,
	"patch_skill": true,
	"write_tool":  true,
	"patch_tool":  true,
}

func truncateWriteArgs(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) != nil {
		return argsJSON
	}
	const omitted = "[omitted after successful write — already applied on disk; read_files to inspect]"
	for _, field := range []string{"content", "old_string", "new_string"} {
		if _, ok := m[field]; ok {
			m[field] = omitted
		}
	}
	if targets, ok := m["targets"].([]any); ok {
		for _, t := range targets {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			for _, field := range []string{"old_string", "new_string"} {
				if _, ok := tm[field]; ok {
					tm[field] = omitted
				}
			}
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return argsJSON
	}
	return string(out)
}

var checkpointClearableTool = map[string]bool{
	"list_files":   true,
	"glob_files":   true,
	"search_files": true,
	"run_command":  true,
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
			if strings.HasPrefix(cached, "data:image/") {
				cachedContent = fmt.Sprintf("[%s] image loaded", toolName)
				slots[i].isImage = true
				slots[i].imageURL = cached
			}
			slots[i].state = slotCached
			slots[i].preMsg = cachedContent
			continue
		}

		if exec.StubTools[toolName] || activatedInBatch[toolName] {
			if exec.StubTools[toolName] {
				activateArgs, _ := json.Marshal(map[string]any{"query": "select:" + toolName})
				if _, err := toolRegister.Dispatch(ctx, exec, "search_tools", activateArgs); err != nil {
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

		if !allowAll && toolNeedsConfirmation(exec, toolName, toolArg, *turnAllowAll) {
			proceed := true
			reason := ""
			if runtime.HasListener(sessionData.ID) {
				reply, err := runtime.Ask(ctx, runtime.Request{
					Kind:      runtime.KindToolConfirm,
					SessionID: sessionData.ID,
					ToolName:  toolName,
					ToolArgs:  toolArg,
				})
				if err != nil {
					proceed = false
				} else {
					proceed = reply.Approve
					reason = reply.Reason
					if reply.Approve && reply.Remember && !sudo.IsActive() {
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
					if cs.isImage {
						injectImageToUserInput(sessionData, cs.imageURL)
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
		if toolRegister.IsFireAndForget(s.name) {
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
			if s.isImage {
				injectImageToUserInput(sessionData, s.imageURL)
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

		if (s.name == "write_file" || s.name == "patch_file") && s.execErr == "" {
			invalidateReadFileCache(alreadyCall, s.args)
		}
		if s.name == "write_todo" && s.execErr == "" && hasCompletedTodo(s.args) {
			todoCheckpointHit = true
		}
		if isWriteLikeTool[s.name] && s.execErr == "" {
			calls[i].Function.Arguments = truncateWriteArgs(calls[i].Function.Arguments)
		}
		alreadyCall[s.hash] = result
		if s.execErr == "" && !strings.HasPrefix(result, "data:image/") && toolcache.IsCacheable(s.name) {
			toolcache.Store(exec.SessionID, s.id, s.name, s.args, result)
		}

		events <- agentTypes.Event{
			Type:     agentTypes.EventToolResult,
			ToolName: s.name,
			ToolID:   s.id,
			Result:   result,
		}

		toolMsgContent := strings.TrimSpace(fmt.Sprintf("[%s] %s", s.name, result))
		if strings.HasPrefix(result, "data:image/") {
			toolMsgContent = fmt.Sprintf("[%s] image loaded", s.name)
			injectImageToUserInput(sessionData, result)
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

func runToolExec(ctx context.Context, exec *toolTypes.Executor, s *toolSlot, events chan<- agentTypes.Event) {
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
