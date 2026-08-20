package file

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func globBatch(ctx context.Context, e *toolTypes.Executor, queries []findQuery) (string, error) {
	seen := make(map[string]struct{})
	var merged []go_pkg_filesystem_reader.File
	for _, q := range queries {
		matches, err := globOne(ctx, e, q.Dir, q.Pattern)
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

	slices.SortFunc(merged, func(a, b go_pkg_filesystem_reader.File) int {
		return strings.Compare(a.Path, b.Path)
	})
	raw, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}

func isCatchAllPattern(pattern string) bool {
	for _, r := range pattern {
		if r != '*' && r != '/' && r != '.' && r != '?' {
			return false
		}
	}
	return true
}

func globOne(ctx context.Context, e *toolTypes.Executor, dir, pattern string) ([]go_pkg_filesystem_reader.File, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if isCatchAllPattern(pattern) {
		return nil, fmt.Errorf("pattern %q matches every file and narrows nothing; use mode=list to see what a directory holds, or mode=search to find files by their content", pattern)
	}

	dir = strings.TrimSpace(dir)
	absPath, err := go_pkg_filesystem.AbsPath(e.WorkDir, dir, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: AbsPath: %w", err)
	}

	if parent, ok := denied.Hit(e.SessionID, absPath); ok {
		return nil, fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	matches, err := go_pkg_filesystem_reader.GlobFiles(absPath, pattern)
	if err != nil {
		if denied.IsPermission(err) {
			denied.Register(e.SessionID, absPath)
			return nil, fmt.Errorf("permission denied: %s (recorded; further reads under this path will be skipped)", absPath)
		}
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: GlobFiles: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}
