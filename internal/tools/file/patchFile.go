package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
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
	before := content
	var skipped []int
	for _, i := range order {
		updated, err := applyTarget(content, targets[i], absPath)
		if updated == content && err == nil {
			skipped = append(skipped, i)
		}
		if err != nil {
			if conflict := batchConflict(before, content, absPath, targets, order, i); conflict != "" {
				return "", fmt.Errorf("targets[%d]: %s", i, conflict)
			}
			return "", fmt.Errorf("targets[%d]: %w", i, err)
		}
		content = updated
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
		sort.Ints(skipped)
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

func applyTarget(content string, target patchTarget, absPath string) (string, error) {
	old := target.OldString
	new := target.NewString
	if old == "" {
		return "", fmt.Errorf("old_string is required")
	}
	if old == new {
		return content, nil
	}
	if !strings.Contains(content, old) {
		return "", anchorNotFound(content, old, absPath)
	}

	search := old
	if new == "" && !strings.HasSuffix(old, "\n") && strings.Contains(content, old+"\n") {
		search = old + "\n"
	}

	if target.ReplaceAll {
		return strings.ReplaceAll(content, search, new), nil
	}
	if rows := rowsOf(content, search); len(rows) > 1 {
		return "", fmt.Errorf("%s occurs on rows %v of %s; extend old_string until it matches once, or set replace_all", old, rows, absPath)
	}
	return strings.Replace(content, search, new, 1), nil
}

func batchConflict(original, current, absPath string, targets []patchTarget, order []int, at int) string {
	old := targets[at].OldString
	if old == "" || strings.Contains(current, old) || !strings.Contains(original, old) {
		return ""
	}

	var by []string
	for _, j := range order {
		if j == at {
			break
		}
		other := targets[j].OldString
		if other != "" && (strings.Contains(old, other) || strings.Contains(other, old)) {
			by = append(by, fmt.Sprintf("targets[%d]", j))
		}
	}
	culprit := "an earlier target in this same call"
	if len(by) > 0 {
		culprit = strings.Join(by, " and ")
	}

	return fmt.Sprintf("%q is still present in %s on disk, but %s rewrote that region earlier in this same call, so the anchor was already gone when this target ran. Nothing was written. Re-reading the file shows the anchor and produces this same failure — merge the overlapping targets into one target, or send them in separate calls", old, absPath, culprit)
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
