package variant

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func removeToolDir(ctx context.Context, e *toolTypes.Executor, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required when mode=remove")
	}

	dir := filepath.Join(filesystem.ScriptToolsDir, name)
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return "", fmt.Errorf("tool %q does not exist", name)
	}

	trashPath, err := filesystem.TrashDir(dir, filesystem.ScriptToolTrashDir, name)
	if err != nil {
		return "", err
	}
	recordRemoval(ctx, e, dir, trashPath, "edit_tool")

	return fmt.Sprintf("trashed: %s", dir), nil
}
