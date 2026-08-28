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

func HasSchedule(name string) bool {
	return go_pkg_filesystem_reader.IsDir(filesystem.ScheduleSkillDir(name))
}

func ReadSchedule(name string) (string, error) {
	path := filesystem.ScheduleSkillPath(name)
	if !go_pkg_filesystem_reader.Exists(path) {
		return "", fmt.Errorf("schedule skill [%s] not found", name)
	}

	result, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem ReadText [%s]: %w", path, err)
	}
	return result, nil
}

type Schedule struct {
	Name        string
	Description string
	Body        string
}

func LoadSchedule(name string) (Schedule, error) {
	raw, err := ReadSchedule(name)
	if err != nil {
		return Schedule{}, err
	}

	out := Schedule{
		Name: name,
		Body: strings.TrimSpace(bodyRegex.ReplaceAllString(raw, "")),
	}
	if header, _, err := getFront([]byte(raw)); err == nil {
		out.Description = getDescription(header)
	}
	return out, nil
}

func WriteSchedule(name, description, body string) error {
	var sb strings.Builder
	sb.WriteString("---\nname: ")
	sb.WriteString(name)
	sb.WriteString("\n")
	if description = strings.TrimSpace(description); description != "" {
		sb.WriteString("description: ")
		sb.WriteString(description)
		sb.WriteString("\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(body))
	sb.WriteString("\n")
	content := sb.String()

	dir := filesystem.ScheduleSkillDir(name)
	if err := go_pkg_filesystem.CheckDir(dir, true); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem CheckDir [%s]: %w", dir, err)
	}

	path := filesystem.ScheduleSkillPath(name)
	if err := go_pkg_filesystem.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem WriteFile [%s]: %w", path, err)
	}
	return nil
}

func GetSchedule(name string) (string, error) {
	result, err := ReadSchedule(name)
	if err != nil {
		return "", err
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
		slog.Debug("historyStore.RecordDelete",
			slog.String("path", dir),
			slog.String("error", err.Error()))
	}
	return nil
}
