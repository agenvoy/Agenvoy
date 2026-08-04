package webui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

const (
	Container  = "open-webui"
	InstallURL = "https://agenvoy.com/scripts/webui.sh"
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

func HostPort(engine string) string {
	if engine == "" {
		return ""
	}
	const format = "{{range $p, $conf := .NetworkSettings.Ports}}{{if $conf}}{{(index $conf 0).HostPort}} {{end}}{{end}}"
	raw, err := exec.Command(engine, "container", "inspect", "-f", format, Container).Output()
	if err != nil {
		return ""
	}
	mapped, _, _ := strings.Cut(strings.TrimSpace(string(raw)), " ")
	return mapped
}

func Port() string {
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.WebuiPort != "" {
		return cfg.WebuiPort
	}
	port := HostPort(Engine())
	if port == "" {
		return ""
	}
	if err := SavePort(port); err != nil {
		return port
	}
	return port
}

func SavePort(port string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	if cfg == nil || cfg.WebuiPort == port {
		return nil
	}
	cfg.WebuiPort = port
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	return nil
}

func URL() string {
	port := Port()
	if port == "" {
		return ""
	}
	return "http://127.0.0.1:" + port
}
