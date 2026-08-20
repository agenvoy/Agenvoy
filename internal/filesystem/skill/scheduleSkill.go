package skill

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func GetSchedule(name string) (string, error) {
	path := filesystem.ScheduleSkillPath(name)
	if !go_pkg_filesystem_reader.Exists(path) {
		return "", fmt.Errorf("schedule skill [%s] not found", name)
	}

	result, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader ReadText [%s]: %w", path, err)
	}
	return strings.TrimSpace(bodyRegex.ReplaceAllString(result, "")), nil
}

func TrashSchedule(ctx context.Context, name string, meta historyStore.Meta) error {
	dir := filesystem.ScheduleSkillDir(name)
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return nil
	}

	trashPath, err := filesystem.TrashDir(dir, filesystem.ScheduleSkillTrashDir, name)
	if err != nil {
		return err
	}

	meta.Tool = "schedules"
	if err := historyStore.RecordDelete(ctx, dir, trashPath, meta); err != nil {
		slog.Warn("historyStore.RecordDelete",
			slog.String("path", dir),
			slog.String("error", err.Error()))
	}
	return nil
}
