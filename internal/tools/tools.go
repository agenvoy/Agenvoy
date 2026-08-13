package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/sudo"
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

	if sudo.IsActive() {
		if blocked, hit := sudo.HitFloor(abs); hit {
			return "", fmt.Errorf("access denied (floor): %s", blocked)
		}
	} else {
		for _, dir := range filesystem.DeniedMap.Dirs {
			needle := "/" + dir
			if strings.Contains(abs, needle+"/") || strings.HasSuffix(abs, needle) {
				return "", fmt.Errorf("access denied: %s", dir)
			}
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
	var moved, failed, unrecorded []string
	for _, arg := range args {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("moveToTrash cancelled: %w", err)
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}

		src, err := go_pkg_filesystem.AbsPath(e.WorkDir, arg, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", arg, err))
			continue
		}

		dst, err := filesystem.MoveToStoreTemp(src)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", arg, err))
			continue
		}

		moved = append(moved, src)
		if err := historyStore.RecordDelete(ctx, src, dst, historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: "rm"}); err != nil {
			slog.Warn("historyStore.RecordDelete",
				slog.String("path", src),
				slog.String("error", err.Error()))
			unrecorded = append(unrecorded, fmt.Sprintf("%s now at %s (%v)", src, dst, err))
		}
	}

	switch {
	case len(moved) == 0 && len(failed) == 0:
		return "", fmt.Errorf("rm requires a path to remove")
	case len(moved) == 0:
		return "", fmt.Errorf("rm removed nothing: %s", strings.Join(failed, "; "))
	}

	report := fmt.Sprintf("moved to %s: %s", filesystem.StoreTempDir, strings.Join(moved, ", "))
	if len(failed) > 0 {
		report += "\nnot removed: " + strings.Join(failed, "; ")
	}
	if len(unrecorded) > 0 {
		report += "\nnot recorded, so restore_file_history will not find these: " + strings.Join(unrecorded, "; ")
	}
	return report, nil
}
