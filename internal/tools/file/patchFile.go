package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
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

	if conflict := insertedAnchor(targets); conflict != "" {
		return "", fmt.Errorf("%s", conflict)
	}

	before := content
	spans, skipped, err := planTargets(content, targets, absPath)
	if err != nil {
		return "", err
	}
	for _, one := range slices.Backward(spans) {
		content = content[:one.start] + one.text + content[one.end:]
	}
	if content == before {
		return fmt.Sprintf("no write: every target in %s already matches its new_string, so the file is unchanged", absPath), nil
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

	note := ""
	if len(skipped) > 0 {
		slices.Sort(skipped)
		note = fmt.Sprintf("\n%d of %d targets were already applied and were skipped: %v", len(skipped), len(targets), skipped)
	}
	return fmt.Sprintf("successfully updated %s", absPath) + note + unrecorded, nil
}

func rejectElided(content string, targets []patchTarget) error {
	values := []string{content}
	for _, one := range targets {
		values = append(values, one.OldString, one.NewString)
	}
	if slices.ContainsFunc(values, toolTypes.IsElided) {
		return fmt.Errorf("%s is a history placeholder, not file content — the earlier write already landed on disk; read_files the file and copy the real text", toolTypes.Elided)
	}
	return nil
}

type patchTarget struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

func insertedAnchor(targets []patchTarget) string {
	for i, one := range targets {
		old := strings.TrimRight(one.OldString, "\n")
		if old == "" {
			continue
		}
		for j := range i {
			if !strings.Contains(targets[j].NewString, old) {
				continue
			}
			return fmt.Sprintf("targets[%d]: %q also occurs inside targets[%d].new_string. Every anchor resolves against the bytes already on disk, so this target cannot reach text another target inserts. Nothing was written — merge the two targets into one, or send them in separate calls", i, old, j)
		}
	}
	return ""
}

type patchSpan struct {
	target int
	start  int
	end    int
	text   string
}

func planTargets(content string, targets []patchTarget, absPath string) ([]patchSpan, []int, error) {
	var spans []patchSpan
	var skipped []int

	for i, one := range targets {
		old := one.OldString
		if old == "" {
			return nil, nil, fmt.Errorf("targets[%d]: old_string is required", i)
		}
		if old == one.NewString {
			skipped = append(skipped, i)
			continue
		}
		if !strings.Contains(content, old) {
			return nil, nil, fmt.Errorf("targets[%d]: %w", i, anchorNotFound(content, old, absPath))
		}

		search := old
		if one.NewString == "" && !strings.HasSuffix(old, "\n") && strings.Contains(content, old+"\n") {
			search = old + "\n"
		}

		at := offsetsOf(content, search)
		if len(at) > 1 && !one.ReplaceAll {
			return nil, nil, fmt.Errorf("targets[%d]: %q occurs on rows %v of %s; extend old_string until it matches once, or set replace_all", i, old, rowsAt(content, at), absPath)
		}
		if !one.ReplaceAll {
			at = at[:1]
		}
		for _, pos := range at {
			spans = append(spans, patchSpan{target: i, start: pos, end: pos + len(search), text: one.NewString})
		}
	}

	slices.SortFunc(spans, func(a, b patchSpan) int { return a.start - b.start })
	for k := 1; k < len(spans); k++ {
		if spans[k].start < spans[k-1].end {
			return nil, nil, fmt.Errorf("targets[%d] and targets[%d] both cover row %d of %s, so applying one would destroy the other's anchor. Nothing was written — merge them into one target, or send them in separate calls", spans[k-1].target, spans[k].target, rowAt(content, spans[k].start), absPath)
		}
	}
	return spans, skipped, nil
}

func offsetsOf(content, search string) []int {
	var at []int
	for idx := 0; ; {
		i := strings.Index(content[idx:], search)
		if i < 0 {
			return at
		}
		pos := idx + i
		at = append(at, pos)
		idx = pos + len(search)
	}
}

func rowAt(content string, pos int) int {
	return strings.Count(content[:pos], "\n") + 1
}

func rowsAt(content string, at []int) []int {
	rows := make([]int, 0, len(at))
	for _, pos := range at {
		rows = append(rows, rowAt(content, pos))
	}
	return rows
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
