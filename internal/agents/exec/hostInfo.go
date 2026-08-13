package exec

import (
	"context"
	"os/exec"
	goRuntime "runtime"
	"strings"
	"sync"
	"time"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

const (
	winHomeProbeTimeout = 3 * time.Second
	winHomeFallback     = "/mnt/<drive>/Users/<WinUser>"
)

type hostInfo struct {
	os      string
	wsl     bool
	winHome string
}

var (
	hostOnce sync.Once
	hostFact hostInfo
)

func host() hostInfo {
	hostOnce.Do(func() {
		hostFact.os = goRuntime.GOOS
		tag := utils.WSLTag()
		if tag == "" {
			return
		}
		hostFact.os = goRuntime.GOOS + " (" + tag + ")"
		hostFact.wsl = true
		hostFact.winHome = windowsHome()
	})
	return hostFact
}

func windowsHome() string {
	if path := winHomeFromInterop(); path != "" {
		return path
	}
	return winHomeFromMount()
}

func winHomeFromInterop() string {
	cmdBin, err := exec.LookPath("cmd.exe")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), winHomeProbeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, cmdBin, "/c", "echo %USERPROFILE%").Output()
	if err != nil {
		return ""
	}
	winPath := strings.TrimSpace(string(out))
	if winPath == "" || strings.Contains(winPath, "%USERPROFILE%") {
		return ""
	}

	pathBin, err := exec.LookPath("wslpath")
	if err != nil {
		return ""
	}
	out, err = exec.CommandContext(ctx, pathBin, "-u", winPath).Output()
	if err != nil {
		return ""
	}
	unixPath := strings.TrimSpace(string(out))
	if !go_pkg_filesystem_reader.IsDir(unixPath) {
		return ""
	}
	return unixPath
}

var winSystemProfile = map[string]bool{
	"public":       true,
	"default":      true,
	"default user": true,
	"all users":    true,
	"defaultuser0": true,
}

func winHomeFromMount() string {
	for _, drive := range []string{"c", "d"} {
		dirs, err := go_pkg_filesystem_reader.ListDirs("/mnt/" + drive + "/Users")
		if err != nil {
			continue
		}
		var found string
		for _, one := range dirs {
			if winSystemProfile[strings.ToLower(one.Name)] {
				continue
			}
			if found != "" {
				return ""
			}
			found = one.Path
		}
		if found != "" {
			return found
		}
	}
	return ""
}

func hostNoteSection() string {
	info := host()
	if !info.wsl {
		return ""
	}
	winHome := info.winHome
	if winHome == "" {
		winHome = winHomeFallback
	}
	return strings.ReplaceAll(configs.WSLHost, "{{.WinHome}}", winHome)
}
