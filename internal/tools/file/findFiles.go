package file

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

const maxFindResultBytes = 1 << 17

type sizeBudget struct {
	left    int
	total   int
	dropped int
	order   string
}

func newSizeBudget(order string) *sizeBudget {
	return &sizeBudget{left: maxFindResultBytes, order: order}
}

func entrySize(file go_pkg_filesystem_reader.File) int {
	raw, _ := json.Marshal(file)
	return len(raw)
}

func (b *sizeBudget) take(list []go_pkg_filesystem_reader.File) []go_pkg_filesystem_reader.File {
	b.total += len(list)
	for i := range list {
		size := entrySize(list[i])
		if size > b.left {
			b.dropped += len(list) - i
			return list[:i]
		}
		b.left -= size
	}
	return list
}

func (b *sizeBudget) notice() string {
	if b.dropped == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n[partial result: %d of %d matching entries returned, %s; %d omitted to stay under %d KiB",
		b.total-b.dropped, b.total, b.order, b.dropped, maxFindResultBytes>>10)
	sb.WriteString(". What is here is accurate, only incomplete. To see the rest, narrow dir, make pattern more specific, or tighten file_pattern — re-running this query unchanged truncates identically.]")
	return sb.String()
}

type findQuery struct {
	Dir         string `json:"dir"`
	Pattern     string `json:"pattern"`
	FilePattern string `json:"file_pattern"`
	Recursive   bool   `json:"recursive"`
}

func registFindFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "find_files",
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Locate files: what a directory holds (list), which paths match a name pattern (glob, most recently modified first), which files contain a string (search, grep by RE2 regex; paged).
Use for 找檔案 / 這個目錄有什麼 / 哪個檔案有這段, and for list_files / glob_files / search_files / grep.
A path you are unsure of comes from here, never from a guess. Contents → read_files; past versions → file_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "glob", "search"},
					"description": "list: entries of each dir. glob: paths matching a filename pattern. search: files whose contents match a regex. Omitted: pattern + file_pattern → search, pattern alone → glob, neither → list.",
					"default":     "list",
				},
				"output": map[string]any{
					"type":        "string",
					"enum":        []string{"files", "content"},
					"description": "mode=search only. files: each matching path with its match count and the row numbers of its first 5 matches — start here, and read_files around those rows directly when the query was precise. content: the matching lines with their text, for when the rows alone cannot tell which match matters.",
					"default":     "files",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "mode=search only: entries to skip — files for output=files, matching lines for output=content. Use the offset the page notice gives; keep queries and output unchanged between pages.",
					"default":     0,
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "mode=search only: entries per page (files or matching lines, per output).",
					"default":     defaultSearchLimit,
				},
				"multiline": map[string]any{
					"type":        "boolean",
					"description": "mode=search only: match the regex against whole files so '.' crosses newlines ('^'/'$' still anchor lines) — for spans like 'type \\w+ struct \\{.*?Name'. Every row a match spans is reported as a matching line.",
					"default":     false,
				},
				"context": map[string]any{
					"type":        "integer",
					"description": "mode=search, output=content: lines shown before and after each match, marked context=true; they do not count toward offset/limit.",
					"default":     0,
				},
				"queries": map[string]any{
					"type":        "array",
					"description": "Every directory and pattern in one call rather than repeated calls; glob and search merge and deduplicate their matches.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"dir": map[string]any{
								"type":        "string",
								"description": "Directory to work in — '.', '~/Desktop', '/abs/path'.",
								"default":     ".",
							},
							"pattern": map[string]any{
								"type":        "string",
								"description": "mode=glob: filename glob relative to dir — '**/*.go', '*.md'; no leading '/' or '~', and it must carry a literal (all-wildcard is rejected). mode=search: RE2 regex matched per line — 'func\\s+\\w+Handler', 'TODO:'.",
							},
							"file_pattern": map[string]any{
								"type":        "string",
								"description": "mode=search: glob narrowing which files to scan — '**/*.go', 'configs/**/*.json'.",
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
				Mode      string      `json:"mode"`
				Output    string      `json:"output"`
				Offset    int         `json:"offset"`
				Limit     int         `json:"limit"`
				Context   int         `json:"context"`
				Multiline bool        `json:"multiline"`
				Queries   []findQuery `json:"queries"`
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
				output := strings.ToLower(strings.TrimSpace(params.Output))
				if output == "" {
					output = "files"
				}
				return searchBatch(ctx, e, params.Queries, output, params.Offset, params.Limit, params.Context, params.Multiline)
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
