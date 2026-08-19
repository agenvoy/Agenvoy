package file

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

type findQuery struct {
	Dir         string `json:"dir"`
	Pattern     string `json:"pattern"`
	FilePattern string `json:"file_pattern"`
	Recursive   bool   `json:"recursive"`
}

func registFindFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "find_files",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `
Locate files three ways: what a directory holds (list), which paths match a name pattern (glob), which files match a string in their content (search — grep by RE2 regex).
Never guess a full path — find it here first, then read_files on what comes back.
Batch every directory and pattern into one 'queries' call; glob and search merge and deduplicate their matches.
Replaces list_files, glob_files and search_files: a skill or instruction naming any of those means this tool — list_files → mode=list, glob_files → mode=glob, search_files → mode=search.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "glob", "search"},
					"description": "list: entries of each dir. glob: paths matching a filename pattern. search: files whose contents match a regex. Omitted: pattern with file_pattern selects search, pattern alone selects glob, neither selects list.",
					"default":     "list",
				},
				"queries": map[string]any{
					"type":        "array",
					"description": "One or more lookups, run together.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"dir": map[string]any{
								"type":        "string",
								"description": "Directory to work in (e.g. '.', '~/Desktop', '/abs/path'). Defaults to the current working directory.",
								"default":     ".",
							},
							"pattern": map[string]any{
								"type":        "string",
								"description": "mode=glob: filename glob relative to dir ('**/*.go', '*.md'), no leading '/' or '~', and it must carry a literal — all-wildcard patterns are rejected. mode=search: RE2 regex matched per line ('func\\s+\\w+Handler', 'TODO:').",
							},
							"file_pattern": map[string]any{
								"type":        "string",
								"description": "mode=search: glob narrowing which files to scan (e.g. '**/*.go', 'configs/**/*.json').",
								"default":     "**/*",
							},
							"recursive": map[string]any{
								"type":        "boolean",
								"description": "mode=list: walk the subtree instead of immediate children.",
								"default":     false,
							},
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
				Mode    string      `json:"mode"`
				Queries []findQuery `json:"queries"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Queries) == 0 {
				return "", fmt.Errorf("queries is required")
			}

			mode := strings.ToLower(strings.TrimSpace(params.Mode))
			if mode == "" {
				mode = inferMode(params.Queries)
			}

			switch mode {
			case "list":
				return listBatch(ctx, e, params.Queries)
			case "glob":
				if err := requirePattern(params.Queries, mode); err != nil {
					return "", err
				}
				return globBatch(ctx, e, params.Queries)
			case "search":
				if err := requirePattern(params.Queries, mode); err != nil {
					return "", err
				}
				return searchBatch(ctx, e, params.Queries)
			}
			return "", fmt.Errorf("unknown mode %q; available: list, glob, search", mode)
		},
	})
}

func inferMode(queries []findQuery) string {
	mode := "list"
	for _, q := range queries {
		if strings.TrimSpace(q.Pattern) == "" {
			continue
		}
		if strings.TrimSpace(q.FilePattern) != "" {
			return "search"
		}
		mode = "glob"
	}
	return mode
}

func requirePattern(queries []findQuery, mode string) error {
	for _, q := range queries {
		if strings.TrimSpace(q.Pattern) == "" {
			return fmt.Errorf("every query needs a pattern when mode=%s", mode)
		}
	}
	return nil
}
