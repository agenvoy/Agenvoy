package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime"
)

type ExecProcessDone struct{}

func (t TUI) runExecProcess(id string, req runtime.Request) (TUI, tea.Cmd) {
	if req.ExecProcess == nil || strings.TrimSpace(req.ExecProcess.Command) == "" {
		runtime.Resolve(id, runtime.Reply{Error: fmt.Errorf("ExecProcess payload missing or empty command")})
		return t, nil
	}

	cmd := exec.Command(req.ExecProcess.Command, req.ExecProcess.Args...)
	hint := fmt.Sprintf("⎯ running: %s %s", req.ExecProcess.Command, strings.Join(req.ExecProcess.Args, " "))

	return t, tea.Sequence(
		tea.Println(hintStyle.Render(hint)+"\n"),
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			reply := runtime.Reply{}
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					reply.ExitCode = exitErr.ExitCode()
				} else {
					reply.ExitCode = -1
					reply.Error = err
				}
			}
			runtime.Resolve(id, reply)
			return ExecProcessDone{}
		}),
	)
}
