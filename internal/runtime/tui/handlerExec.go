package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

const toolBufDisplayCap = 4

type agentEvent struct {
	event agentTypes.Event
}

func (t *TUI) collapseToolBuf() tea.Cmd {
	count := t.toolCount
	t.toolBuf = nil
	t.toolCount = 0
	if count == 0 {
		return nil
	}
	label := "tool call"
	if count != 1 {
		label = "tool calls"
	}
	return tea.Println("\n" + hintStyle.Render(fmt.Sprintf("  Ran %d %s", count, label)))
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

	for ev := range ch {
		send(agentEvent{event: ev})
		switch ev.Type {
		case agentTypes.EventDone, agentTypes.EventReasoning, agentTypes.EventText, agentTypes.EventToolCall, agentTypes.EventCompact:
			time.Sleep(10 * time.Millisecond)
		}
	}

	err := <-done
	send(agentExecDone{err: err})
}

func (t TUI) handleAgentEvent(ev agentTypes.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case agentTypes.EventAgentSelect:
		if ev.Source == "" {
			t.activity = "selecting agent…"
		}

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

	case agentTypes.EventToolCall:
		if ev.ToolName != "" && ev.ToolName != "ask_user" && ev.ToolName != "store_secret" &&
			ev.ToolName != "write_todo" {
			t.activity = "tool: " + ev.ToolName
			line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, "")
			if ok {
				t.toolCount++
				t.toolBuf = append(t.toolBuf, line)
				if len(t.toolBuf) > toolBufDisplayCap {
					t.toolBuf = t.toolBuf[len(t.toolBuf)-toolBufDisplayCap:]
				}
			}
			return t, nil
		}

	case agentTypes.EventReasoning:
		line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, "")
		if !ok {
			return t, nil
		}
		collapse := t.collapseToolBuf()
		if collapse != nil {
			return t, tea.Sequence(collapse, tea.Println("\n"+line))
		}
		return t, tea.Println("\n" + line)

	case agentTypes.EventToolResult:
		t.activity = ""
		return t, nil

	case agentTypes.EventSummaryGenerate:
		t.activity = "summarizing…"

	case agentTypes.EventCompact:
		if ev.Text == "history" {
			t.activity = "compacting history…"
		} else {
			t.activity = "compacting tool history…"
		}
		line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, "")
		if ok {
			t.toolBuf = append(t.toolBuf, line)
		}
		return t, nil

	case agentTypes.EventText:
		if ev.Source == "" {
			collapse := t.collapseToolBuf()
			raw := ev.Text

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
		collapse := t.collapseToolBuf()
		t.todos = nil
		if ev.Usage != nil {
			t.tokens = ev.Usage.Input + ev.Usage.Output
			t.lastIn = ev.Usage.Input
			t.lastOut = ev.Usage.Output
			t.lastCacheRead = ev.Usage.CacheRead
		}
		finishedAt := time.Now().Format("2006-01-02 15:04:05")
		if collapse != nil {
			line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, finishedAt)
			if !ok {
				return t, collapse
			}
			return t, tea.Sequence(collapse, tea.Println(line))
		}
		line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, finishedAt)
		if !ok {
			return t, nil
		}
		return t, tea.Println(line)

	case agentTypes.EventUsageUpdate:
		if ev.Source == "" && ev.Usage != nil {
			t.lastIn = ev.Usage.Input
			t.lastOut = ev.Usage.Output
			t.lastCacheRead = ev.Usage.CacheRead
		}
		return t, nil

	}

	line, ok := renderAgentEvent(ev, t.runTarget, t.cwd, t.width, "")
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
