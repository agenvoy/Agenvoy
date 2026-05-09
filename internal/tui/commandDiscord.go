package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type DiscordDone struct {
	action string
	err    error
}

func (t TUI) commandDiscord(action string) (TUI, tea.Cmd, bool) {
	self, err := os.Executable()
	if err != nil {
		return t, tea.Println("\n" + errorStyle.Render(fmt.Sprintf("[!] os.Executable: %v", err))), true
	}

	cmd := exec.Command(self, "discord", action)
	cmd.Env = os.Environ()

	exec := tea.ExecProcess(cmd, func(err error) tea.Msg {
		return DiscordDone{action: action, err: err}
	})

	return t, tea.Sequence(
		tea.Println("\n"+hintStyle.Render(fmt.Sprintf("⎯ discord %s · ctrl+c to cancel", action))),
		exec,
	), true
}
