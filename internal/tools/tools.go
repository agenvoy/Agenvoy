package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"

	"github.com/pardnchiu/agenvoy/configs"
)

var (
	TUIOnlyTools  []string
	TUIOnlySkills []string
)

func init() {
	var data struct {
		Tools  []string `json:"tools"`
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal(configs.TUITools, &data); err != nil {
		return
	}
	TUIOnlyTools = data.Tools
	TUIOnlySkills = data.Skills
}

var WorkDirChangeHook func(path string)

func changeWorkDir(e *toolTypes.Executor, args []string) (string, error) {
	var positional []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
		}
	}
	if len(positional) > 1 {
		return "", fmt.Errorf("cd accepts at most one positional argument, got %d", len(positional))
	}

	var target string
	switch {
	case len(positional) == 0 || positional[0] == "~":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("os.UserHomeDir: %w", err)
		}
		target = home
	case strings.HasPrefix(positional[0], "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("os.UserHomeDir: %w", err)
		}
		target = filepath.Join(home, positional[0][2:])
	default:
		target = positional[0]
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(e.WorkDir, target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("filepath.Abs: %w", err)
	}
	abs = filepath.Clean(abs)

	for _, dir := range filesystem.DeniedMap.Dirs {
		needle := "/" + dir
		if strings.Contains(abs, needle+"/") || strings.HasSuffix(abs, needle) {
			return "", fmt.Errorf("access denied: %s. %s", dir, deniedHint)
		}
	}

	if !go_pkg_filesystem_reader.IsDir(abs) {
		return "", fmt.Errorf("not a directory or does not exist: %s", abs)
	}

	e.WorkDir = abs
	if WorkDirChangeHook != nil {
		WorkDirChangeHook(abs)
	}
	return fmt.Sprintf("Changed working directory to: %s", abs), nil
}

func moveToTrash(ctx context.Context, e *toolTypes.Executor, args []string) (string, error) {
	return file.RemoveToTrash(ctx, e, args, "rm")
}
