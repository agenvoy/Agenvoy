package variant

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func writeSkillFile(ctx context.Context, e *toolTypes.Executor, path, content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content is required when mode=write")
	}

	absPath, err := skillPath(path)
	if err != nil {
		return "", err
	}

	change := capture(absPath)
	if err := go_pkg_filesystem.WriteFile(absPath, content, 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/agenvoy/internal/filesystem: WriteFile [%s]: %w", absPath, err)
	}
	record(ctx, e, change, content, "edit_skill")

	return fmt.Sprintf("created: %s", absPath), nil
}

func skillPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	absPath := filepath.Clean(filepath.Join(filesystem.SkillsDir, path))
	if !strings.HasPrefix(absPath, filesystem.SkillsDir+string(filepath.Separator)) {
		return "", fmt.Errorf("path must stay within skills dir")
	}
	return absPath, nil
}
