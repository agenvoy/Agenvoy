package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registOpenFile() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "open_file",
		AlwaysAllow: true,
		Description: "Open a file with the OS default application (e.g. play a video, view an image, open a PDF viewer). Use this instead of run_command's `open`/`xdg-open` — the sandboxed run_command path cannot reach the OS's app-launch service. Only for launching a GUI viewer; not for reading file contents (use read_files for that).",
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
			return openFile(ctx, e, params.Path)
		},
	})
}

func openFile(ctx context.Context, e *toolTypes.Executor, path string) (string, error) {
	baseDir := e.WorkDir
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

	if parent, ok := denied.Hit(e.SessionID, absPath); ok {
		return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	var binary string
	switch runtime.GOOS {
	case "darwin":
		binary = "open"
	case "linux":
		binary = "xdg-open"
	case "windows":
		binary = "explorer"
	default:
		return "", fmt.Errorf("open_file: unsupported platform %s", runtime.GOOS)
	}

	cmd := exec.CommandContext(ctx, binary, absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if denied.IsPermission(err) {
			denied.Register(e.SessionID, absPath)
		}
		return fmt.Sprintf("%s\nError: %s", string(output), err.Error()), nil
	}

	return fmt.Sprintf("opened %s", absPath), nil
}
