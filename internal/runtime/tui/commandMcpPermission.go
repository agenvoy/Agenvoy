package tui

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	allowTool "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/tool"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

func allowAllEntry(server string) string {
	return mcpToolPrefix(server) + "*"
}

func mcpToolPrefix(server string) string {
	return "mcp__" + server + "__"
}

type McpPermissionResult struct {
	server string
	tools  []mcp.Tool
	err    error
}

type McpPermissionPick struct {
	server  string
	entries []string
}

func (t TUI) openMcpPermission(name string) (TUI, tea.Cmd) {
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tools, err := mcp.Manager().Tools(ctx, name)
		return McpPermissionResult{server: name, tools: tools, err: err}
	}
}

func (t TUI) runMcpPermissionResult(msg McpPermissionResult) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s tools: %v", msg.server, msg.err)) + "\n")
	}
	if len(msg.tools) == 0 {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s exposes no tools", msg.server)) + "\n")
	}

	names := make([]string, 0, len(msg.tools))
	for _, tool := range msg.tools {
		if trimmed := strings.TrimSpace(tool.Name); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	sort.Strings(names)

	granted := allowTool.LoadGlobal()
	all := allowAllEntry(msg.server)

	options := make([]string, 0, len(names)+1)
	values := make([]string, 0, len(names)+1)
	multi := make(map[int]bool, len(names)+1)

	options = append(options, "all · every tool of this server")
	values = append(values, all)
	if granted[all] {
		multi[0] = true
	}

	for _, name := range names {
		entry := mcpToolPrefix(msg.server) + name
		options = append(options, name)
		values = append(values, entry)
		if granted[entry] {
			multi[len(values)-1] = true
		}
	}

	if multi[0] {
		for i := 1; i <= len(names); i++ {
			multi[i] = true
		}
	} else if len(names) > 0 {
		multi[0] = true
		for i := 1; i <= len(names); i++ {
			if !multi[i] {
				multi[0] = false
				break
			}
		}
	}

	server := msg.server
	t.popup = &Popup{
		kind:     popupMultiSelect,
		title:    "MCP · " + server + " · permission",
		subtitle: "selected tools skip the confirmation prompt in every work directory",
		options:  options,
		values:   values,
		multi:    multi,
		onToggle: syncAllRow,
		onConfirm: func(chosen string) any {
			pick := McpPermissionPick{server: server}
			if chosen != "" {
				pick.entries = strings.Split(chosen, "\x1F")
			}
			return pick
		},
	}
	return t, nil
}

func syncAllRow(p *Popup, index int) {
	if len(p.options) < 2 {
		return
	}

	if index == 0 {
		for i := 1; i < len(p.options); i++ {
			p.multi[i] = p.multi[0]
		}
		return
	}

	for i := 1; i < len(p.options); i++ {
		if !p.multi[i] {
			p.multi[0] = false
			return
		}
	}
	p.multi[0] = true
}

func (t TUI) runMcpPermissionPick(msg McpPermissionPick) (TUI, tea.Cmd) {
	entries := msg.entries
	if slices.Contains(entries, allowAllEntry(msg.server)) {
		entries = []string{allowAllEntry(msg.server)}
	}

	if err := allowTool.ReplaceGlobalPrefix(mcpToolPrefix(msg.server), entries); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s permission: %v", msg.server, err)) + "\n")
	}

	if len(entries) == 0 {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s: every tool now asks for confirmation", msg.server)) + "\n")
	}
	if entries[0] == allowAllEntry(msg.server) {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s: all tools always allowed", msg.server)) + "\n")
	}
	return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s: %d tool(s) always allowed", msg.server, len(entries))) + "\n")
}
