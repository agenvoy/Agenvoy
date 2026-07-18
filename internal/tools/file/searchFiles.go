package file

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func registSearchFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "search_files",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
Search file contents by RE2 regex within a directory.
Locate code or text when the matching string is known but the file is not.
Scope with file_pattern glob (e.g. '**/*.go', 'configs/**').
Supports multiple searches in one call — when several patterns/dirs need searching, put them all in ` + "`queries`" + ` rather than issuing separate calls; matches are merged and deduplicated.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"queries": map[string]any{
					"type":        "array",
					"description": "One or more content searches.",
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
								"description": "RE2 regex matched per line (e.g. 'func\\s+\\w+Handler', 'TODO:', 'api_key').",
							},
							"file_pattern": map[string]any{
								"type":        "string",
								"description": "Glob relative to dir to narrow files (e.g. '**/*.go', 'configs/**/*.json').",
								"default":     "**/*",
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
					Dir         string `json:"dir"`
					Pattern     string `json:"pattern"`
					FilePattern string `json:"file_pattern"`
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
				pattern := strings.TrimSpace(q.Pattern)
				if pattern == "" {
					return "", fmt.Errorf("pattern is required")
				}

				dir := strings.TrimSpace(q.Dir)
				absPath, err := go_pkg_filesystem.AbsPath(e.WorkDir, dir, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
				if err != nil {
					return "", fmt.Errorf("go_pkg_filesystem.AbsPath: %w", err)
				}

				if parent, ok := denied.Hit(e.SessionID, absPath); ok {
					return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
				}

				var filePatterns []string
				if q.FilePattern != "" {
					filePatterns = strings.Split(filepath.ToSlash(q.FilePattern), "/")
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
						return "", fmt.Errorf("permission denied: %s (recorded; further reads under this path will be skipped)", absPath)
					}
					return "", fmt.Errorf("go_pkg_filesystem_reader.SearchFiles: %w", err)
				}
				if err := ctx.Err(); err != nil {
					return "", err
				}

				for _, m := range matches {
					if rel, err := filepath.Rel(absPath, m.Path); err == nil {
						m.Path = rel
					}
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
		},
	})
}
