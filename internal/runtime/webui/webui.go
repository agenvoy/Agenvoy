package webui

import (
	"os/exec"
	"strings"
)

const (
	Container  = "open-webui"
	Port       = "17990"
	InstallURL = "https://agenvoy.com/scripts/webui.sh"
	URL        = "http://127.0.0.1:" + Port
)

func Engine() string {
	for _, name := range []string{"podman", "docker"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func Status(engine string) (exists, running bool) {
	if engine == "" {
		return false, false
	}
	raw, err := exec.Command(engine, "container", "inspect", "-f", "{{.State.Running}}", Container).Output()
	if err != nil {
		return false, false
	}
	return true, strings.TrimSpace(string(raw)) == "true"
}
