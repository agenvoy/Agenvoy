package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func registListFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_files",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
List directory entries.
Use to inspect immediate children of a directory; recursive=true walks subtree files.
For finding files by name or pattern, prefer glob_files instead.
Batch multiple directories into one 'dirs' call instead of separate calls.
Returns a JSON object mapping each requested dir to its entries (or an error string for that dir).`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dirs": map[string]any{
					"type":        "array",
					"description": "One or more directories to list.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"dir": map[string]any{
								"type":        "string",
								"description": "Directory to list (e.g. '.', '~/Desktop', '/abs/path'). Defaults to current working directory.",
								"default":     "",
							},
							"recursive": map[string]any{
								"type":        "boolean",
								"description": "Walk subtree files. Default false.",
								"default":     false,
							},
						},
					},
				},
			},
			"required": []string{
				"dirs",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			var params struct {
				Dirs []struct {
					Dir       string `json:"dir"`
					Recursive bool   `json:"recursive"`
				} `json:"dirs"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Dirs) == 0 {
				return "", fmt.Errorf("dirs is required")
			}

			out := make(map[string]any, len(params.Dirs))
			for _, d := range params.Dirs {
				files, err := listOne(ctx, e, d.Dir, d.Recursive)
				if err != nil {
					out[d.Dir] = "error: " + err.Error()
					continue
				}
				out[d.Dir] = files
				if err := ctx.Err(); err != nil {
					return "", err
				}
			}

			raw, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(raw), nil
		},
	})
}

func listOne(ctx context.Context, e *toolTypes.Executor, dir string, recursive bool) ([]go_pkg_filesystem_reader.File, error) {
	dir = strings.TrimSpace(dir)
	absPath, err := go_pkg_filesystem.AbsPath(e.WorkDir, dir, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: AbsPath: %w", err)
	}

	if parent, ok := denied.Hit(e.SessionID, absPath); ok {
		return nil, fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	if file, err := os.Open(absPath); err != nil {
		if denied.IsPermission(err) {
			denied.Register(e.SessionID, absPath)
			return nil, fmt.Errorf("permission denied: %s (recorded; further reads under this path will be skipped)", absPath)
		}
		return nil, fmt.Errorf("os.Open: %w", err)
	} else {
		file.Close()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if recursive {
		files, err := go_pkg_filesystem_reader.WalkFiles(absPath, go_pkg_filesystem_reader.ListOption{
			SkipExcluded:      true,
			SkipDenied:        true,
			IgnoreWalkError:   true,
			IncludeNonRegular: true,
		})
		if err != nil {
			return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: WalkFiles: %w", err)
		}
		return files, nil
	}

	files, err := go_pkg_filesystem_reader.ListAll(absPath, go_pkg_filesystem_reader.ListOption{SkipExcluded: true})
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: ListAll: %w", err)
	}
	return files, nil
}
