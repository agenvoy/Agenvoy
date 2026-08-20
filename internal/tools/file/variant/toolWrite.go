package variant

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func toolTarget(name, tag string, mustExist bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	switch tag {
	case "json", "script":
		dir := filepath.Join(filesystem.ScriptToolsDir, name)
		if mustExist && !go_pkg_filesystem_reader.IsDir(dir) {
			return "", fmt.Errorf("tool %q does not exist", name)
		}
		if tag == "json" {
			return filepath.Join(dir, "tool.json"), nil
		}
		return filepath.Join(dir, "script.py"), nil
	case "api":
		target := filepath.Join(filesystem.APIToolsDir, name+".json")
		if mustExist && !go_pkg_filesystem_reader.Exists(target) {
			return "", fmt.Errorf("api tool %q does not exist", name)
		}
		return target, nil
	}
	return "", fmt.Errorf("tag must be 'json', 'script', or 'api', got %q", tag)
}

func writeToolFile(ctx context.Context, e *toolTypes.Executor, name, tag, content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content is required when mode=write")
	}

	target, err := toolTarget(name, tag, false)
	if err != nil {
		return "", err
	}

	change := capture(target)
	if err := go_pkg_filesystem.WriteFile(target, content, 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/agenvoy/internal/filesystem: WriteFile [%s]: %w", target, err)
	}
	record(ctx, e, change, content, "edit_tool")

	return fmt.Sprintf("created: %s", target), nil
}
