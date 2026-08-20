package file

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func searchBatch(ctx context.Context, e *toolTypes.Executor, queries []findQuery) (string, error) {
	seen := make(map[string]struct{})
	var merged []go_pkg_filesystem_reader.File
	for _, q := range queries {
		matches, err := searchOne(ctx, e, q.Dir, q.Pattern, q.FilePattern)
		if err != nil {
			return "", err
		}
		for _, m := range matches {
			if _, ok := seen[m.Path]; ok {
				continue
			}
			seen[m.Path] = struct{}{}
			merged = append(merged, m)
		}
	}

	if len(merged) == 0 {
		return "no files found", nil
	}

	slices.SortFunc(merged, func(a, b go_pkg_filesystem_reader.File) int {
		return strings.Compare(a.Path, b.Path)
	})
	raw, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}

func searchOne(ctx context.Context, e *toolTypes.Executor, dir, pattern, filePattern string) ([]go_pkg_filesystem_reader.File, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	dir = strings.TrimSpace(dir)
	absPath, err := go_pkg_filesystem.AbsPath(e.WorkDir, dir, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: AbsPath: %w", err)
	}

	if parent, ok := denied.Hit(e.SessionID, absPath); ok {
		return nil, fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	var filePatterns []string
	if filePattern != "" {
		filePatterns = strings.Split(filepath.ToSlash(filePattern), "/")
	}
	matches, err := go_pkg_filesystem_reader.SearchFiles(absPath, pattern, filePatterns, 0,
		go_pkg_filesystem_reader.ListOption{
			SkipExcluded:    true,
			SkipDenied:      true,
			IgnoreWalkError: true,
		})
	if err != nil {
		if denied.IsPermission(err) {
			denied.Register(e.SessionID, absPath)
			return nil, fmt.Errorf("permission denied: %s (recorded; further reads under this path will be skipped)", absPath)
		}
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: SearchFiles: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for i, m := range matches {
		if rel, err := filepath.Rel(absPath, m.Path); err == nil {
			matches[i].Path = rel
		}
	}
	return matches, nil
}
