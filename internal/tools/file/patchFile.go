package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func patchFileTargets(ctx context.Context, e *toolTypes.Executor, path0 string, targets []patchTarget) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("targets is required when mode=patch")
	}

	baseDir := e.WorkDir
	if baseDir == "" {
		baseDir = filesystem.DownloadDir
	}

	path := strings.TrimSpace(path0)
	absPath, err := boundary.Resolve(e.SessionID, baseDir, path)
	if err != nil {
		return "", fmt.Errorf("boundary.Resolve: %w", err)
	}
	if absPath == "" {
		return "", fmt.Errorf("path or name is required")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("os.Stat: %w", err)
	}
	if info.Size() > maxReadSize {
		return "", fmt.Errorf("file too large (%d bytes, max 1 MB)", info.Size())
	}

	content, err := go_pkg_filesystem.ReadText(absPath)
	if err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: ReadText: %w", err)
	}
	change := historyStore.CaptureContent(absPath, content)

	order := make([]int, len(targets))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ra, rb := targets[order[a]].Row, targets[order[b]].Row
		if ra > 0 && rb > 0 {
			return ra > rb
		}
		return ra > 0 && rb == 0
	})

	before := content
	for _, i := range order {
		updated, err := applyTarget(content, targets[i], absPath)
		if err != nil {
			return "", fmt.Errorf("targets[%d]: %w", i, err)
		}
		content = updated
	}
	if strings.TrimSpace(content) == "" && strings.TrimSpace(before) != "" {
		return "", fmt.Errorf("every target applied but %s would be left empty (it holds %d bytes); nothing written — re-read the file and narrow the anchors, or call write_file if emptying it is the intent", absPath, len(before))
	}

	if err := go_pkg_filesystem.WriteFile(absPath, content, 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}

	e.RecordFile(absPath)

	var unrecorded string
	if err := historyStore.Record(ctx, change, historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: "edit_file"}); err != nil {
		slog.Debug("historyStore.Record",
			slog.String("path", absPath),
			slog.String("error", err.Error()))
		unrecorded = fmt.Sprintf("\nthe previous version was not recorded (%v), so this edit cannot be undone", err)
	}

	return fmt.Sprintf("successfully updated %s", absPath) + unrecorded, nil
}

type patchTarget struct {
	OldString    string `json:"old_string"`
	NewString    string `json:"new_string"`
	ReplaceAll   bool   `json:"replace_all"`
	InsertString string `json:"insert_string"`
	Row          int    `json:"row"`
}

func applyTarget(content string, target patchTarget, absPath string) (string, error) {
	switch {
	case target.InsertString != "":
		if target.OldString != "" {
			return "", fmt.Errorf("insert_string cannot be combined with old_string")
		}
		if target.Row <= 0 {
			return "", fmt.Errorf("row is required when insert_string is set")
		}
		return insertAtRow(content, target.InsertString, target.Row)

	case target.OldString != "":
		old := target.OldString
		new := target.NewString
		if old == new {
			return "", fmt.Errorf("no edit needed")
		}
		if !strings.Contains(content, old) {
			return "", anchorNotFound(content, old, absPath)
		}

		search := old
		if new == "" && !strings.HasSuffix(old, "\n") && strings.Contains(content, old+"\n") {
			search = old + "\n"
		}

		switch {
		case target.ReplaceAll:
			return strings.ReplaceAll(content, search, new), nil
		case target.Row > 0:
			return replaceAtRow(content, search, new, target.Row)
		default:
			if rows := rowsOf(content, search); len(rows) > 1 {
				return "", fmt.Errorf("%s occurs on rows %v of %s; set row to one of them or replace_all", old, rows, absPath)
			}
			return strings.Replace(content, search, new, 1), nil
		}

	default:
		return "", fmt.Errorf("either old_string or insert_string is required")
	}
}

func anchorNotFound(content, old, absPath string) error {
	head := ""
	for line := range strings.SplitSeq(old, "\n") {
		if strings.TrimSpace(line) != "" {
			head = strings.TrimSpace(line)
			break
		}
	}

	lines := strings.Split(content, "\n")
	var near []string
	if head != "" {
		for i, line := range lines {
			if strings.TrimSpace(line) == head || strings.Contains(line, head) {
				near = append(near, fmt.Sprintf("row %d is %q", i+1, line))
				if len(near) == 5 {
					break
				}
			}
		}
	}

	if len(near) == 0 {
		return fmt.Errorf("%q is not found in %s and nothing there resembles it; the file holds %d lines — re-read it and build the anchor from its current bytes", old, absPath, len(lines))
	}
	return fmt.Errorf("%q is not found in %s, but %s — copy the anchor from those exact bytes, whitespace included", old, absPath, strings.Join(near, "; "))
}

func replaceAtRow(content, search, new string, row int) (string, error) {
	idx := 0
	for {
		i := strings.Index(content[idx:], search)
		if i < 0 {
			break
		}
		pos := idx + i
		line := strings.Count(content[:pos], "\n") + 1
		if line == row {
			return content[:pos] + new + content[pos+len(search):], nil
		}
		idx = pos + 1
	}
	return "", fmt.Errorf("no match for %q at row %d; it is on rows %v", search, row, rowsOf(content, search))
}

func rowsOf(content, search string) []int {
	var rows []int
	for idx := 0; ; {
		i := strings.Index(content[idx:], search)
		if i < 0 {
			return rows
		}
		pos := idx + i
		rows = append(rows, strings.Count(content[:pos], "\n")+1)
		idx = pos + 1
	}
}

func insertAtRow(content, insert string, row int) (string, error) {
	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	if row < 1 || row > lineCount+1 {
		return "", fmt.Errorf("row %d out of range (file has %d lines)", row, lineCount)
	}

	insert = strings.TrimSuffix(insert, "\n")
	insert = strings.TrimSuffix(insert, "\r")

	idx := row - 1
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:idx]...)
	out = append(out, strings.Split(insert, "\n")...)
	out = append(out, lines[idx:]...)
	return strings.Join(out, "\n"), nil
}
