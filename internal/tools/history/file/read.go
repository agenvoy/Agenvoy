package fileHistory

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aymanbagabas/go-udiff"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func read(ctx context.Context, e *toolTypes.Executor, paths []string) (string, error) {
	blocks := make([]string, 0, len(paths))
	for _, raw := range paths {
		path, err := absPath(e, raw)
		if err != nil {
			blocks = append(blocks, fmt.Sprintf("%s\n%v", raw, err))
			continue
		}

		list, err := historyStore.Newest(ctx, historyStore.Filter{Path: path, Limit: 1})
		if err != nil {
			return "", fmt.Errorf("internal/runtime/history: Newest: %w", err)
		}
		if len(list) == 0 {
			blocks = append(blocks, fmt.Sprintf("%s\nno recorded changes", path))
			continue
		}
		blocks = append(blocks, describe(ctx, list[0]))
	}
	return strings.Join(blocks, "\n\n"), nil
}

func describe(ctx context.Context, row historyStore.Row) string {
	path := filepath.Join(row.Dir, row.Name)
	header := fmt.Sprintf("%s — %s by %s at %s",
		path, row.Action, row.Tool, time.Unix(0, row.ChangedAt).Format(historyStore.TimeLayout))

	if reason := row.RestoreBlock(); reason != "" {
		return fmt.Sprintf("%s\ncannot be shown or restored: %s", header, reason)
	}

	_, content, err := historyStore.Get(ctx, row.ID)
	if err != nil {
		return fmt.Sprintf("%s\nunreadable: %v", header, err)
	}
	if row.Action == historyStore.ActionDelete {
		text, err := go_pkg_filesystem.ReadText(row.TrashPath)
		if err != nil {
			return fmt.Sprintf("%s\nunreadable from %s: %v", header, row.TrashPath, err)
		}
		content = []byte(text)
		header += fmt.Sprintf("\nthe file now sits in %s", row.TrashPath)
	}
	if len(content) == 0 && row.Action == historyStore.ActionModify && row.TrashPath != "" {
		return fmt.Sprintf("%s\n%d bytes, kept as a copy at %s — too large to hold in the history, still restorable from there",
			header, row.Size, row.TrashPath)
	}
	if row.Action == historyStore.ActionCreate {
		header += "\nthis is the version the file was created with; restoring to before it deletes the file"
	}

	stored := string(content)
	switch {
	case bytes.ContainsRune(content, 0):
		return fmt.Sprintf("%s\nbinary content, %d bytes — not shown", header, len(content))
	case !go_pkg_filesystem_reader.Exists(path):
		return fmt.Sprintf("%s\nnothing at that path now. stored version:\n\n%s", header, stored)
	case !go_pkg_filesystem_reader.IsFile(path):
		return fmt.Sprintf("%s\na directory sits at that path now, so there is nothing to compare", header)
	}

	current, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return fmt.Sprintf("%s\ncannot read the file on disk: %v", header, err)
	}

	switch {
	case stored == current:
		return fmt.Sprintf("%s\nidentical to the file on disk now", header)
	case strings.ContainsRune(current, 0):
		return fmt.Sprintf("%s\nthe file on disk is binary, %d bytes, against %d bytes recorded — not shown",
			header, len(current), len(stored))
	}
	return fmt.Sprintf("%s\ndiff from that version to the file on disk now:\n\n%s",
		header, udiff.Unified("recorded", "current", stored, current))
}
