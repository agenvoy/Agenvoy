package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// not suitable using popup, direct using cmd
type ModelAddDone struct {
	err error
}

func (t TUI) commandModelAdd() (TUI, tea.Cmd, bool) {
	self, err := os.Executable()
	if err != nil {
		return t, tea.Println("\n" + errorStyle.Render(fmt.Sprintf("[!] os.Executable: %v", err))), true
	}

	cmd := exec.Command(self, "model", "add")
	cmd.Env = os.Environ()

	exec := tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return ModelAddDone{err: err}
		}
		return ModelAddDone{}
	})

	return t, tea.Sequence(
		tea.Println("\n"+hintStyle.Render("⎯ launching add-model flow · ctrl+c to cancel")),
		exec,
	), true
}
