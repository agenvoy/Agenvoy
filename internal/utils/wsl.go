package utils

import (
	"os"
	goRuntime "runtime"
	"strings"
	"sync"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

var (
	wslOnce sync.Once
	wslKind string
)

func WSLTag() string {
	wslOnce.Do(func() {
		if goRuntime.GOOS != "linux" {
			return
		}
		if raw, err := go_pkg_filesystem.ReadText("/proc/sys/kernel/osrelease"); err == nil {
			lower := strings.ToLower(raw)
			switch {
			case strings.Contains(lower, "wsl2"):
				wslKind = "WSL2"
				return
			case strings.Contains(lower, "microsoft"):
				wslKind = "WSL1"
				return
			}
		}
		if raw, err := go_pkg_filesystem.ReadText("/proc/version"); err == nil {
			if strings.Contains(strings.ToLower(raw), "microsoft") {
				wslKind = "WSL"
				return
			}
		}
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			wslKind = "WSL"
		}
	})
	return wslKind
}

func IsWSL() bool {
	return WSLTag() != ""
}

func WSLChromePath() string {
	if !IsWSL() {
		return ""
	}
	for _, path := range []string{
		"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
		"/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe",
		"/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
	} {
		if go_pkg_filesystem_reader.Exists(path) {
			return path
		}
	}
	return ""
}
