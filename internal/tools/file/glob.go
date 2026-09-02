package file

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
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
	budget := newSizeBudget()
	merged = budget.take(merged)
	raw, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw) + budget.notice(), nil
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
	absPath, err := boundary.Resolve(e.SessionID, e.WorkDir, dir)
	if err != nil {
		return nil, fmt.Errorf("boundary.Resolve: %w", err)
	}

	matches, err := go_pkg_filesystem_reader.GlobFiles(absPath, pattern)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: GlobFiles: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}
