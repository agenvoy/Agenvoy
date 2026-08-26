package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

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

	target := "~"
	if len(positional) == 1 {
		target = positional[0]
	}
	abs := go_pkg_utils.AbsPath(e.WorkDir, target)

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
