package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

type McpMenuPick struct {
	value string
}

type McpServerAction struct {
	server string
	action string
}

type McpReconnectDone struct {
	server string
	err    error
}

type McpToolsResult struct {
	server string
	tools  []mcp.Tool
	err    error
}

func (t TUI) commandMcp(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		if parts[1] == "add" {
			return t.commandMcpAdd()
		}
		if _, ok := mcpServerStatus(parts[1]); ok {
			next, cmd := t.openMcpServerMenu(parts[1])
			return next, cmd, true
		}
	}

	list := mcpStatusList()
	options := make([]string, 0, len(list)+1)
	values := make([]string, 0, len(list)+1)
	tails := make([]string, 0, len(list)+1)

	maxName, maxTransport := 0, 0
	for _, s := range list {
		maxName = max(maxName, lipgloss.Width(s.Name))
		maxTransport = max(maxTransport, lipgloss.Width(s.Transport))
	}
	for _, s := range list {
		options = append(options, padToWidth(s.Name, maxName+2)+padToWidth(s.Transport, maxTransport+1))
		values = append(values, "server:"+s.Name)
		tails = append(tails, mcpStateLabel(s))
	}
	options = append(options, "add")
	values = append(values, "add")
	tails = append(tails, "")

	t.popup = &Popup{
		kind:       popupSingleSelect,
		title:      "MCP",
		options:    options,
		optionTail: tails,
		values:     values,
		maxVisible: cmdSelectorMaxVisible,
		onConfirm: func(chosen string) any {
			return McpMenuPick{value: chosen}
		},
	}
	return t, nil, true
}

func mcpStatusList() []mcp.ServerInfo {
	m := mcp.Manager()
	if m == nil {
		return nil
	}
	return m.Status("")
}

func mcpServerStatus(name string) (mcp.ServerInfo, bool) {
	for _, s := range mcpStatusList() {
		if s.Name == name {
			return s, true
		}
	}
	return mcp.ServerInfo{}, false
}

func mcpStateLabel(s mcp.ServerInfo) string {
	switch {
	case !s.Connected:
		return errorStyle.Render("[disconnected]")
	case s.Error != "":
		return warnStyle.Render("[tools unavailable]")
	default:
		return okayStyle.Render("[connected]")
	}
}

func (t TUI) openMcpServerMenu(name string) (TUI, tea.Cmd) {
	info, ok := mcpServerStatus(name)
	if !ok {
		return t, tea.Println(msgError(fmt.Sprintf("mcp server %q not found", name)) + "\n")
	}

	subtitle := fmt.Sprintf("%s  %s", info.Transport, mcpStateLabel(info))
	if info.Error != "" {
		subtitle = fmt.Sprintf("%s\n%s", subtitle, errorStyle.Render(info.Error))
	}

	values := []string{"tools", "permission", "reconnect", "remove"}
	details := []string{"list registered tools", "pick always-allowed tools", "", ""}

	cfg, err := mcp.Load()
	if err == nil && cfg.Servers[name].IsOAuth() {
		values = append([]string{"login", "client"}, values...)
		details = append([]string{"browser oauth", "set oauth client id / secret"}, details...)
	}
	options := optionColumn(values, details)

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "MCP  " + name,
		subtitle: subtitle,
		options:  options,
		values:   values,
		onConfirm: func(chosen string) any {
			return McpServerAction{server: name, action: chosen}
		},
	}
	return t, nil
}

func (t TUI) runMcpMenuPick(value string) (TUI, tea.Cmd) {
	if value == "add" {
		next, cmd, _ := t.commandMcpAdd()
		return next, cmd
	}
	if name, ok := strings.CutPrefix(value, "server:"); ok {
		return t.openMcpServerMenu(name)
	}
	return t, nil
}

func (t TUI) runMcpServerAction(msg McpServerAction) (TUI, tea.Cmd) {
	switch msg.action {
	case "login":
		return t.startMcpLogin(msg.server)
	case "client":
		return t.openMcpClientID(msg.server)
	case "remove":
		return t.runMcpRemove(msg.server)
	case "reconnect":
		return t.reconnectMcpServer(msg.server)
	case "tools":
		return t.listMcpTools(msg.server)
	case "permission":
		return t.openMcpPermission(msg.server)
	}
	return t, nil
}

func (t TUI) reconnectMcpServer(name string) (TUI, tea.Cmd) {
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return McpReconnectDone{server: name, err: mcp.Manager().ReconnectServer(ctx, name)}
	}
}

func (t TUI) listMcpTools(name string) (TUI, tea.Cmd) {
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tools, err := mcp.Manager().Tools(ctx, name)
		return McpToolsResult{server: name, tools: tools, err: err}
	}
}

func (t TUI) runMcpToolsResult(msg McpToolsResult) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("%s tools: %v", msg.server, msg.err)) + "\n")
	}
	if len(msg.tools) == 0 {
		return t, tea.Println(msgLog(fmt.Sprintf("%s exposes no tools", msg.server)) + "\n")
	}

	maxName := 0
	for _, tool := range msg.tools {
		maxName = max(maxName, len(tool.Name))
	}
	lines := make([]string, 0, len(msg.tools)+1)
	lines = append(lines, msgLog(fmt.Sprintf("%s  %d tools", msg.server, len(msg.tools))))
	for _, tool := range msg.tools {
		summary, _, _ := strings.Cut(strings.TrimSpace(tool.Description), "\n")
		lines = append(lines, fmt.Sprintf("  %s  %s",
			whiteStyle.Render(fmt.Sprintf("%-*s", maxName, tool.Name)),
			hintStyle.Render(summary)))
	}
	return t, tea.Println(strings.Join(lines, "\n") + "\n")
}
