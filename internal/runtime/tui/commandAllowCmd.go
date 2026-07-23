package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	allowCmd "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/cmd"
)

type AllowCmdSubmit struct {
	name string
}

func (t TUI) commandAllowCmd(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) >= 2 {
		name := strings.TrimSpace(strings.Join(parts[1:], " "))
		next, cmd := t.runAllowCmdAppend(name)
		return next, cmd, true
	}
	t.popup = &Popup{
		kind:  popupText,
		title: "Command to allow (appended to config.json white_list)",
		input: newPopupInput("", false),
		onConfirm: func(value string) any {
			return AllowCmdSubmit{name: strings.TrimSpace(value)}
		},
	}
	return t, nil, true
}

func (t TUI) runAllowCmdAppend(name string) (TUI, tea.Cmd) {
	if strings.TrimSpace(name) == "" {
		return t, tea.Println(errorStyle.Render("[!] command name required") + "\n")
	}
	added, err := allowCmd.Append(name)
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] allow-cmd: %v", err)) + "\n")
	}
	if !added {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s already allowed", name)) + "\n")
	}
	return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ added to config.white_list: %s · restart daemon (agen stop) to apply", name)) + "\n")
}
