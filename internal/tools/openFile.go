package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func registOpenFile() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "open_file",
		AlwaysAllow: true,
		Description: `Hands a file to the OS default application — plays a video, shows an image, opens a PDF viewer.
Use for 打開 / 播放 / 看一下這個檔案, and for open / xdg-open / start.
run_command cannot do this: its sandbox has no route to the app-launch service. Reading the contents instead → read_files.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File to open (e.g. '/abs/path/video.mp4', '~/Downloads/report.pdf', 'relative/file.png').",
				},
			},
			"required": []string{"path"},
		},
		Timeout: 15 * time.Second,
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			return OpenFile(ctx, e.WorkDir, e.SessionID, params.Path)
		},
	})
}

func OpenFile(ctx context.Context, workDir, sessionID, path string) (string, error) {
	baseDir := workDir
	if baseDir == "" {
		baseDir = filesystem.DownloadDir
	}

	absPath, err := go_pkg_filesystem.AbsPath(baseDir, path, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err != nil {
		return "", fmt.Errorf("go_pkg_filesystem.AbsPath: %w", err)
	}
	if absPath == "" {
		return "", fmt.Errorf("path is required")
	}

	if parent, ok := denied.Hit(sessionID, absPath); ok {
		return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", absPath)
	case "linux":
		bin, args, cmdErr := linuxOpenCmd(ctx, absPath)
		if cmdErr != nil {
			return "", fmt.Errorf("open_file: %w", cmdErr)
		}
		cmd = exec.CommandContext(ctx, bin, args...)
	case "windows":
		cmd = exec.CommandContext(ctx, "explorer", absPath)
	default:
		return "", fmt.Errorf("open_file: unsupported platform %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if denied.IsPermission(err) {
			denied.Register(sessionID, absPath)
		}
		return fmt.Sprintf("%s\nError: %s", string(output), err.Error()), nil
	}

	return fmt.Sprintf("opened %s", absPath), nil
}

func linuxOpenCmd(ctx context.Context, target string) (string, []string, error) {
	if utils.IsWSL() {
		if winPath, err := wslToWindowsPath(ctx, target); err == nil {
			if bin, lookErr := exec.LookPath("cmd.exe"); lookErr == nil {
				return bin, []string{"/c", "start", "", winPath}, nil
			}
		}
	}
	if bin, err := exec.LookPath("wslview"); err == nil {
		return bin, []string{target}, nil
	}
	if bin, err := exec.LookPath("xdg-open"); err == nil {
		return bin, []string{target}, nil
	}
	return "", nil, fmt.Errorf("no opener available (cmd.exe/wslview/xdg-open not found)")
}

func wslToWindowsPath(ctx context.Context, path string) (string, error) {
	bin, err := exec.LookPath("wslpath")
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, bin, "-w", path).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
