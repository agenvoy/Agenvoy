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

func removeSkillDir(ctx context.Context, e *toolTypes.Executor, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required when mode=remove")
	}

	dir := filepath.Join(filesystem.SkillsDir, name)
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return "", fmt.Errorf("skill %q does not exist", name)
	}

	trashPath, err := filesystem.TrashDir(dir, filesystem.SkillTrashDir, name)
	if err != nil {
		return "", err
	}
	recordRemoval(ctx, e, dir, trashPath, "edit_skill")

	return fmt.Sprintf("trashed: %s", dir), nil
}
