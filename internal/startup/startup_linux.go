package startup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

const (
	unitName    = "agenvoy.service"
	execTimeout = 10 * time.Second
)

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.UserHomeDir", "", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", unitName), nil
}

func Enabled() bool {
	_, err := run("systemctl", "--user", "is-enabled", unitName)
	return err == nil
}

func Enable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.Executable", "", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.UserHomeDir", "", err)
	}
	path, err := unitPath()
	if err != nil {
		return "", err
	}
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.CheckDir", filepath.Dir(path), err)
	}
	if err := go_pkg_filesystem.WriteFile(path, unit(exe, home), 0644); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", path, err)
	}
	if _, err := run("systemctl", "--user", "daemon-reload"); err != nil {
		return "", err
	}
	if _, err := run("systemctl", "--user", "enable", unitName); err != nil {
		return "", err
	}
	record(true)
	return path + " · takes effect at next login", nil
}

func Disable() (string, error) {
	path, err := unitPath()
	if err != nil {
		return "", err
	}
	if !go_pkg_filesystem_reader.Exists(path) {
		record(false)
		return "already off", nil
	}
	if _, err := run("systemctl", "--user", "disable", unitName); err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.Remove", path, err)
	}
	if _, err := run("systemctl", "--user", "daemon-reload"); err != nil {
		return "", err
	}
	record(false)
	return path + " removed · running daemon untouched", nil
}

func unit(exe, home string) string {
	var sb strings.Builder
	sb.WriteString("[Unit]\nDescription=Agenvoy daemon\n\n")
	sb.WriteString("[Service]\nType=simple\n")
	fmt.Fprintf(&sb, "ExecStart=%q --daemon\n", escape(exe))
	fmt.Fprintf(&sb, "WorkingDirectory=%s\n", escape(home))
	if path := os.Getenv("PATH"); path != "" {
		fmt.Fprintf(&sb, "Environment=%q\n", "PATH="+escape(path))
	}
	fmt.Fprintf(&sb, "StandardOutput=append:%s\n", escape(filesystem.DaemonLogPath))
	fmt.Fprintf(&sb, "StandardError=append:%s\n\n", escape(filesystem.DaemonLogPath))
	sb.WriteString("[Install]\nWantedBy=default.target\n")
	return sb.String()
}

func escape(str string) string {
	return strings.ReplaceAll(str, "%", "%%")
}

func run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	raw, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	out := strings.TrimSpace(string(raw))
	if err != nil {
		return out, utilsmsg.New(utilsmsg.Internal, name+" "+strings.Join(args, " "), "%w: %s", err, out)
	}
	return out, nil
}
