package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type agentEvent struct {
	event agentTypes.Event
}

func (t *TUI) collapseToolBuf() tea.Cmd {
	count, subs := t.toolCount, t.subCount
	t.toolBuf = nil
	t.toolCount = 0
	t.subCount = 0
	if count == 0 && subs == 0 {
		return nil
	}
	var parts []string
	if count > 0 {
		label := "tool call"
		if count != 1 {
			label += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", count, label))
	}
	if subs > 0 {
		label := "subagent call"
		if subs != 1 {
			label += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", subs, label))
	}
	return tea.Println("\n" + hintStyle.Render("  Ran "+strings.Join(parts, " · ")))
}

const subagentLogLines = 3

const commandLogLines = 128

type subagentLive struct {
	tools int
	lines []string
}

func (b *subagentLive) render(name string, width int) string {
	header := "  ⎿ [" + name + "]"
	if b.tools > 0 {
		label := "tool"
		if b.tools != 1 {
			label = "tools"
		}
		header += fmt.Sprintf(" %d %s", b.tools, label)
	}
	var sb strings.Builder
	sb.WriteString(hintStyle.Render(header))
	for _, line := range b.lines {
		sb.WriteByte('\n')
		sb.WriteString(hintStyle.Render(go_pkg_utils.TruncateString("    ⎿  "+line, max(width-4, 32))))
	}
	return sb.String()
}

func (t *TUI) trackSubagent(name, activity string) *subagentLive {
	if t.subBuf == nil {
		t.subBuf = map[string]*subagentLive{}
	}
	block, ok := t.subBuf[name]
	if !ok {
		block = &subagentLive{}
		t.subBuf[name] = block
		t.subOrder = append(t.subOrder, name)
	}
	if activity != "" {
		block.lines = append(block.lines, activity)
		if len(block.lines) > subagentLogLines {
			block.lines = block.lines[len(block.lines)-subagentLogLines:]
		}
	}
	return block
}

func toolActivity(ev agentTypes.Event, cwd string) string {
	label := utils.ToolName(ev.ToolName)
	if arg := utils.FormatToolArgs(ev.ToolName, ev.ToolArgs, cwd); arg != "" {
		label += "(" + arg + ")"
	}
	return oneLine(label)
}

func (t *TUI) dropSubagent(name string) {
	if t.subBuf == nil {
		return
	}
	delete(t.subBuf, name)
	if i := slices.Index(t.subOrder, name); i >= 0 {
		t.subOrder = slices.Delete(t.subOrder, i, i+1)
	}
}

type agentExec struct {
	cancel context.CancelFunc
}

type agentExecDone struct {
	err error
}

func runExec(parentCtx context.Context, input string, allowAll bool, workDir, sessionID, pendingTask, historyContent string) {
	ctx, cancel := context.WithCancel(exec.WithDcPushPrefix(parentCtx, go_pkg_utils.TruncateString(input, 32)))
	send(agentExec{cancel: cancel})

	ch := make(chan agentTypes.Event, 16)
	wrapped := wrapEventsPublish(ctx, sessionID, ch)
	done := make(chan error, 1)

	scanner := agents.Scanner()
	if scanner != nil {
		scanner.Scan()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				close(wrapped)
				done <- fmt.Errorf("exec.Run panic: %v", r)
			}
		}()
		err := exec.Run(
			ctx,
			agents.DispatcherBot(),
			agents.Registry(),
			scanner,
			input,
			nil,
			nil,
			wrapped,
			allowAll,
			workDir,
			sessionID,
			pendingTask,
			historyContent,
		)
		close(wrapped)
		done <- err
	}()

	terminated := false
	for ev := range ch {
		if ev.Type == agentTypes.EventTextDelta {
			continue
		}
		switch ev.Type {
		case agentTypes.EventDone, agentTypes.EventCanceled, agentTypes.EventError:
			terminated = true
		}
		send(agentEvent{event: ev})
		switch ev.Type {
		case agentTypes.EventDone, agentTypes.EventReasoning, agentTypes.EventText, agentTypes.EventToolCall, agentTypes.EventCompact:
			time.Sleep(10 * time.Millisecond)
		}
	}

	err := <-done
	if err != nil && !terminated {
		ev := agentTypes.Event{Type: agentTypes.EventError, Text: err.Error()}
		if errors.Is(err, context.Canceled) {
			ev.Type = agentTypes.EventCanceled
		}
		publishEventToDaemon(ctx, sessionID, ev)
	}
	send(agentExecDone{err: err})
}

func (t TUI) handleAgentEvent(ev agentTypes.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case agentTypes.EventAgentSelect:
		if ev.Source != "" {
			t.trackSubagent(ev.Source, "selecting agent…")
			return t, nil
		}
		t.activity = "selecting agent…"

	case agentTypes.EventAgentResult:
		if ev.Source == "" {
			str := strings.TrimSpace(ev.Text)
			t.currentModel = str
			t.activity = str
		}

	case agentTypes.EventTodoUpdate:
		if ev.Source == "" {
			t.todos = ev.Todos
		}
		return t, nil

	case agentTypes.EventUserInjected:
		if ev.Source == "" {
			t.pendingSteer = nil
		}
		line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, "")
		if !ok {
			return t, nil
		}
		collapse := t.collapseToolBuf()
		if collapse != nil {
			return t, tea.Sequence(collapse, tea.Println("\n"+line))
		}
		return t, tea.Println("\n" + line)

	case agentTypes.EventToolCall:
		if utils.HideToolEvent(ev.ToolName, ev.ToolArgs) {
			return t, nil
		}
		if ev.ToolName != "" {
			if ev.Source != "" {
				t.trackSubagent(ev.Source, toolActivity(ev, t.cwd)).tools++
				return t, nil
			}
			t.activity = "tool: " + ev.ToolName
			t.toolLog = nil
			line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, "")
			if ok {
				t.emitted = true
				if utils.IsSubagentInvoke(ev.ToolName, ev.ToolArgs) {
					t.subCount++
					t.subActive++
				} else {
					t.toolCount++
				}
				t.toolBuf = append(t.toolBuf, line)
			}
			return t, nil
		}

	case agentTypes.EventToolProgress:
		if ev.Source != "" {
			t.trackSubagent(ev.Source, ev.ToolName+": "+oneLine(ev.Text))
			return t, nil
		}
		t.emitted = true
		t.activity = "tool: " + ev.ToolName
		t.toolLog = append(t.toolLog, oneLine(ev.Text))
		if len(t.toolLog) > commandLogLines {
			t.toolLog = t.toolLog[len(t.toolLog)-commandLogLines:]
		}
		return t, nil

	case agentTypes.EventToolSkipped:
		if ev.Source != "" {
			t.trackSubagent(ev.Source, "skipped: "+ev.ToolName)
			return t, nil
		}

	case agentTypes.EventReasoning:
		if ev.Source != "" {
			t.trackSubagent(ev.Source, "✻ "+oneLine(toPureText(ev.Text)))
			return t, nil
		}
		line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, "")
		if !ok {
			return t, nil
		}
		t.emitted = true
		collapse := t.collapseToolBuf()
		if collapse != nil {
			return t, tea.Sequence(collapse, tea.Println("\n"+line))
		}
		return t, tea.Println("\n" + line)

	case agentTypes.EventToolResult:
		if ev.Source != "" {
			if ev.ToolName == "subagents" {
				t.dropSubagent(ev.Source)
			}
			return t, nil
		}

		if utils.IsSubagentInvoke(ev.ToolName, ev.ToolArgs) {
			t.subActive = max(t.subActive-1, 0)
			if t.subActive == 0 {
				t.subBuf, t.subOrder = nil, nil
			}
		}
		t.activity = ""
		t.toolLog = nil
		return t, nil

	case agentTypes.EventSummaryGenerate:
		t.activity = "summarizing…"

	case agentTypes.EventCompact:
		if ev.Source != "" {
			t.trackSubagent(ev.Source, "compacting…")
			return t, nil
		}
		if ev.Text == "history" {
			t.activity = "compacting history…"
		} else {
			t.activity = "compacting tool history…"
		}
		line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, "")
		if ok {
			t.toolBuf = append(t.toolBuf, line)
		}
		return t, nil

	case agentTypes.EventText:
		if ev.Source == "" {
			t.emitted = true
			collapse := t.collapseToolBuf()
			raw := utils.StripFileMarkers(ev.Text)

			if len(t.tableBuf) > 0 {
				if strings.Contains(raw, "|") {
					t.tableBuf = append(t.tableBuf, raw)
					if collapse != nil {
						return t, collapse
					}
					return t, nil
				}
				cmds := t.flushTableBuf()
				cmds = append(cmds, t.printStreamLine(renderMarkdown(raw, t.width)))
				if collapse != nil {
					cmds = append([]tea.Cmd{collapse}, cmds...)
				}
				return t, tea.Sequence(cmds...)
			}

			if strings.Contains(raw, "|") {
				t.tableBuf = append(t.tableBuf, raw)
				if collapse != nil {
					return t, collapse
				}
				return t, nil
			}

			if collapse != nil {
				return t, tea.Sequence(collapse, t.printStreamLine(renderMarkdown(raw, t.width)))
			}
			return t, t.printStreamLine(renderMarkdown(raw, t.width))
		}

	case agentTypes.EventTextDone:
		if ev.Source == "" {
			var cmd tea.Cmd
			if len(t.tableBuf) > 0 {
				cmd = tea.Batch(t.flushTableBuf()...)
			}
			t.streaming = false
			return t, cmd
		}
		return t, nil

	case agentTypes.EventDone:
		if ev.Source != "" {
			t.dropSubagent(ev.Source)
			return t, nil
		}
		t.toolLog = nil
		collapse := t.collapseToolBuf()
		t.todos = nil
		t.subBuf, t.subOrder, t.subActive = nil, nil, 0
		if ev.Usage != nil {
			t.tokens = ev.Usage.Input + ev.Usage.Output + ev.Usage.CacheRead + ev.Usage.CacheCreate
			t.lastIn = ev.Usage.Input
			t.lastOut = ev.Usage.Output
			t.lastCacheRead = ev.Usage.CacheRead
			t.lastCacheCreate = ev.Usage.CacheCreate
		}
		finishedAt := time.Now().Format("2006-01-02 15:04:05")
		if collapse != nil {
			line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, finishedAt)
			if !ok {
				return t, collapse
			}
			return t, tea.Sequence(collapse, tea.Println(line))
		}
		line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, finishedAt)
		if !ok {
			return t, nil
		}
		return t, tea.Println(line)

	case agentTypes.EventCanceled:
		if ev.Source != "" {
			t.dropSubagent(ev.Source)
			return t, nil
		}
		t.toolLog = nil
		collapse := t.collapseToolBuf()
		t.todos = nil
		t.subBuf, t.subOrder, t.subActive = nil, nil, 0
		finishedAt := time.Now().Format("2006-01-02 15:04:05")
		line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, finishedAt)
		if !ok {
			return t, collapse
		}
		if collapse != nil {
			return t, tea.Sequence(collapse, tea.Println(line))
		}
		return t, tea.Println(line)

	case agentTypes.EventUsageUpdate:
		if ev.Source == "" && ev.Usage != nil {
			t.lastIn = ev.Usage.Input
			t.lastOut = ev.Usage.Output
			t.lastCacheRead = ev.Usage.CacheRead
			t.lastCacheCreate = ev.Usage.CacheCreate
		}
		return t, nil

	}

	line, ok := renderAgentEvent(t.ctx, true, ev, t.runTarget, t.cwd, t.width, "")
	if !ok {
		return t, nil
	}
	return t, tea.Println(line)
}

func (t *TUI) printStreamLine(line string) tea.Cmd {
	line = wrapText(line, t.width-2)
	var rendered string
	if !t.streaming {
		t.streaming = true
		t.activity = "responding"
		prefix := systemStyle.Render("⏺ ")
		if strings.TrimSpace(t.runTarget) != "" {
			prefix = warnStyle.Render("⏺ [" + t.runTarget + "] ")
		}
		rendered = "\n" + prefix + line
	} else {
		rendered = "  " + line
	}
	return tea.Println(rendered)
}

func (t *TUI) flushTableBuf() []tea.Cmd {
	block := strings.Join(t.tableBuf, "\n")
	t.tableBuf = nil

	rendered := renderTables(block, t.width-2)
	rendered = renderMarkdown(rendered, t.width-2)

	var sb strings.Builder
	for i, line := range strings.Split(rendered, "\n") {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i == 0 && !t.streaming {
			t.streaming = true
			t.activity = "responding"
			sb.WriteByte('\n')
			if strings.TrimSpace(t.runTarget) != "" {
				sb.WriteString(warnStyle.Render("⏺ [" + t.runTarget + "] "))
			} else {
				sb.WriteString(systemStyle.Render("⏺ "))
			}
		} else {
			sb.WriteString("  ")
		}
		sb.WriteString(line)
	}
	return []tea.Cmd{tea.Println(sb.String() + "\n")}
}
