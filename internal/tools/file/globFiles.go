package file

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func registGlobFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "glob_files",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
Find files matching glob patterns within a directory (e.g. '**/*.go').
Use when only a filename or partial path is known — never guess full paths.
Batch multiple patterns/dirs into one 'queries' call instead of separate calls; matches are merged and deduplicated.
Call read_files on each match before editing to confirm the correct file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"queries": map[string]any{
					"type":        "array",
					"description": "One or more glob searches.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"dir": map[string]any{
								"type":        "string",
								"description": "Directory to search in (e.g. '.', '~/downloads', '/abs/path'). Defaults to current working directory.",
								"default":     ".",
							},
							"pattern": map[string]any{
								"type":        "string",
								"description": "Glob pattern relative to dir (e.g. '**/*.go', '*.md'). No leading '/' or '~' — put absolute paths in dir.",
							},
						},
						"required": []string{
							"pattern",
						},
					},
				},
			},
			"required": []string{
				"queries",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			var params struct {
				Queries []struct {
					Dir     string `json:"dir"`
					Pattern string `json:"pattern"`
				} `json:"queries"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Queries) == 0 {
				return "", fmt.Errorf("queries is required")
			}

			seen := make(map[string]struct{})
			var merged []go_pkg_filesystem_reader.File
			for _, q := range params.Queries {
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
		},
	})
}

func globOne(ctx context.Context, e *toolTypes.Executor, dir, pattern string) ([]go_pkg_filesystem_reader.File, error) {
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
