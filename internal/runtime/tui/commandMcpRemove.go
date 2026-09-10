package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

type McpRemovePick struct {
	server string
}

type McpRemoveConfirm struct {
	server string
	yes    bool
}

func (t TUI) openMcpRemoveConfirm(server string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Remove " + server + " ?",
		subtitle: "the server is disconnected and its oauth credentials are cleared",
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		onConfirm: func(chosen string) any {
			return McpRemoveConfirm{server: server, yes: chosen == "yes"}
		},
	}
	return t, nil
}

func (t TUI) runMcpRemove(server string) (TUI, tea.Cmd) {
	cfg, err := mcp.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("mcp.Load: %v", err)) + "\n")
	}
	if _, ok := cfg.Servers[server]; !ok {
		return t, tea.Println(msgError(fmt.Sprintf("mcp server %q not found", server)) + "\n")
	}
	delete(cfg.Servers, server)
	if err := mcp.Save(cfg); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("mcp.Save: %v", err)) + "\n")
	}
	mcp.Manager().Disconnect(server)

	removed := msgLog(fmt.Sprintf("removed: %s", server))
	if err := mcp.ClearOAuth(server); err != nil {
		return t, tea.Println(removed + "\n" + msgWarn(fmt.Sprintf("oauth credentials left in keychain: %v", err)) + "\n")
	}
	return t, tea.Println(removed + "\n")
}
