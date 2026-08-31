package startup

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const label = "com.pardnchiu.agenvoy"

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("os.UserHomeDir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func Enabled() bool {
	path, err := unitPath()
	if err != nil {
		return false
	}
	return go_pkg_filesystem_reader.Exists(path)
}

func Enable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("os.UserHomeDir: %w", err)
	}
	path, err := unitPath()
	if err != nil {
		return "", err
	}
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", fmt.Errorf("go_pkg_filesystem.CheckDir: %w", err)
	}
	if err := go_pkg_filesystem.WriteFile(path, plist(exe, home), 0644); err != nil {
		return "", fmt.Errorf("go_pkg_filesystem.WriteFile: %w", err)
	}
	return path + " · takes effect at next login", nil
}

func Disable() (string, error) {
	path, err := unitPath()
	if err != nil {
		return "", err
	}
	if !go_pkg_filesystem_reader.Exists(path) {
		return "already off", nil
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("os.Remove: %w", err)
	}
	return path + " removed · running daemon untouched", nil
}

func plist(exe, home string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	sb.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	writeString(&sb, "Label", label)
	sb.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	fmt.Fprintf(&sb, "\t\t<string>%s</string>\n", escape(exe))
	sb.WriteString("\t\t<string>--daemon</string>\n")
	sb.WriteString("\t</array>\n")
	sb.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	writeString(&sb, "WorkingDirectory", home)
	if path := os.Getenv("PATH"); path != "" {
		sb.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
		fmt.Fprintf(&sb, "\t\t<key>PATH</key>\n\t\t<string>%s</string>\n", escape(path))
		sb.WriteString("\t</dict>\n")
	}
	writeString(&sb, "StandardOutPath", filesystem.DaemonLogPath)
	writeString(&sb, "StandardErrorPath", filesystem.DaemonLogPath)
	sb.WriteString("</dict>\n</plist>\n")
	return sb.String()
}

func writeString(sb *strings.Builder, key, value string) {
	fmt.Fprintf(sb, "\t<key>%s</key>\n\t<string>%s</string>\n", escape(key), escape(value))
}

func escape(str string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(str))
	return buf.String()
}
