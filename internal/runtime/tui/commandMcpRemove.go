package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

func (t TUI) runMcpRemove(server string) (TUI, tea.Cmd) {
	cfg, err := mcp.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] mcp.Load: %v", err)) + "\n")
	}
	if _, ok := cfg.Servers[server]; !ok {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] mcp server %q not found", server)) + "\n")
	}
	delete(cfg.Servers, server)
	if err := mcp.Save(cfg); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] mcp.Save: %v", err)) + "\n")
	}
	mcp.Manager().Disconnect(server)

	removed := hintStyle.Render(fmt.Sprintf("⎯ removed: %s", server))
	if err := mcp.ClearOAuth(server); err != nil {
		return t, tea.Println(removed + "\n" + warnStyle.Render(fmt.Sprintf("[!] oauth credentials left in keychain: %v", err)) + "\n")
	}
	return t, tea.Println(removed + "\n")
}
