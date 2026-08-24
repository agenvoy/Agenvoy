package tui

import (
	"os/exec"
)

type RestrictedAuthDone struct {
	pendingID string
	cached    bool
	err       error
}

func sudoCached() bool {
	return exec.Command("sudo", "-n", "-v").Run() == nil
}
