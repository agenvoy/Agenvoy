package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type CmdDone struct{ err error }

func (t *TUI) setCmdMode(on bool) {
	t.cmdMode = on
	if on {
		t.textarea.Placeholder = `shell command · enter run · shift+t chat mode · esc cancel`
		t.textarea.SetPromptFunc(2, func(lineIdx int) string {
			if lineIdx == 0 {
				return warnStyle.Render("$ ")
			}
			return "  "
		})
		return
	}
	t.textarea.Placeholder = `/ commands · enter send · alt+enter newline · esc cancel · shift+t cmd mode`
	t.textarea.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return whiteStyle.Render("❯ ")
		}
		return "  "
	})
}

func (t TUI) runShellCmd(content string) (TUI, tea.Cmd) {
	t.execHandoff = true

	cmd := exec.Command("sh", "-c", content)
	cmd.Env = os.Environ()
	cmd.Dir = t.cwd
	return t, tea.Sequence(
		tea.ClearScreen,
		tea.Println(shellEchoBlock(content)),
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			return CmdDone{err: err}
		}),
	)
}
