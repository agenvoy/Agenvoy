package sessionLog

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	tuiHash "github.com/pardnchiu/agenvoy/internal/session/tui"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

const (
	maxActionLogSize = 1 << 20
	trimTargetSize   = 768 << 10
)

func formatActionEvent(event agentTypes.Event) string {
	switch event.Type {
	case agentTypes.EventText:
		str := strings.TrimSpace(event.Text)
		if str == "" {
			return ""
		}
		return withTimestamp("assistant", event.TaskHash, flatten(str))

	case agentTypes.EventReasoning:
		str := strings.TrimSpace(event.Text)
		if str == "" {
			return ""
		}
		return withTimestamp("thinking", event.TaskHash, flatten(str))

	case agentTypes.EventToolCall:
		display := utils.FormatToolEvent(event.ToolName, event.ToolArgs)
		if display == "" {
			return ""
		}
		return withTimestamp("tool_call", event.TaskHash, flatten(display))

	case agentTypes.EventToolResult:
		status := "ok"
		if event.Err != nil {
			status = "err"
		}
		return withTimestamp("tool_result", event.TaskHash, fmt.Sprintf("%s %s", event.ToolName, status))

	case agentTypes.EventTodoUpdate:
		if len(event.Todos) == 0 {
			return ""
		}
		raw, err := json.Marshal(event.Todos)
		if err != nil {
			return ""
		}
		return withTimestamp("todo", event.TaskHash, string(raw))

	case agentTypes.EventToolSkipped:
		return withTimestamp("tool_skipped", event.TaskHash, event.ToolName)

	case agentTypes.EventToolConfirm:
		return withTimestamp("tool_confirm", event.TaskHash, event.ToolName)

	case agentTypes.EventPending:
		return withTimestamp("pending", event.TaskHash, event.Text)

	case agentTypes.EventExecError, agentTypes.EventError:
		body := ""
		if event.Err != nil {
			body = flatten(event.Err.Error())
		} else if event.Text != "" {
			body = flatten(event.Text)
		} else {
			return ""
		}
		if event.ToolName != "" {
			body = fmt.Sprintf("%s %s", event.ToolName, body)
		}
		return withTimestamp("error", event.TaskHash, body)

	case agentTypes.EventFileChanged:
		if len(event.Files) == 0 {
			return ""
		}
		raw, err := json.Marshal(event.Files)
		if err != nil {
			return ""
		}
		return withTimestamp("edited_files", event.TaskHash, string(raw))

	case agentTypes.EventDone:
		parts := []string{event.Model}
		if event.Duration > 0 {
			parts = append(parts, fmt.Sprintf("dur=%s", event.Duration.Round(time.Millisecond)))
		}
		if event.Usage != nil {
			total, hitPct := agentTypes.InputTotals(event.Usage)
			in := fmt.Sprintf("in=%d", total)
			if hitPct > 0 {
				in = fmt.Sprintf("%s (%d%%)", in, hitPct)
			}
			parts = append(parts, in, fmt.Sprintf("out=%d", event.Usage.Output))
		}
		return withTimestamp("done", event.TaskHash, strings.Join(parts, " "))

	case agentTypes.EventCanceled:
		parts := []string{event.Model}
		if event.Duration > 0 {
			parts = append(parts, fmt.Sprintf("dur=%s", event.Duration.Round(time.Millisecond)))
		}
		return withTimestamp("canceled", event.TaskHash, strings.Join(parts, " "))

	case agentTypes.EventSkillResult:
		str := strings.TrimSpace(event.Text)
		if str == "" {
			return ""
		}
		return withTimestamp("skill_result", event.TaskHash, flatten(str))

	case agentTypes.EventAgentResult:
		str := strings.TrimSpace(event.Text)
		if str == "" {
			return ""
		}
		return withTimestamp("agent_result", event.TaskHash, flatten(str))
	}
	return ""
}

func withTimestamp(kind, taskHash, body string) string {
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("[%s][%s][%s][%s] %s", ts, tuiHash.Get(), kind, taskHash, body)
}

func flatten(str string) string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "\r", "\n")
	str = strings.ReplaceAll(str, "\n", ActionNewlineMarker)
	return str
}
