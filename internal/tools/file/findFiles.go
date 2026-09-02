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

const (
	maxFindResultBytes = 100 << 10
	maxMatchesPerFile  = 100
)

type sizeBudget struct {
	left    int
	total   int
	dropped int
	capped  int
}

func newSizeBudget() *sizeBudget {
	return &sizeBudget{left: maxFindResultBytes}
}

func entrySize(file go_pkg_filesystem_reader.File) int {
	raw, _ := json.Marshal(file)
	return len(raw)
}

func (b *sizeBudget) take(list []go_pkg_filesystem_reader.File) []go_pkg_filesystem_reader.File {
	b.total += len(list)
	for i := range list {
		cut := false
		if len(list[i].Matches) > maxMatchesPerFile {
			list[i].Matches = list[i].Matches[:maxMatchesPerFile]
			cut = true
		}
		size := entrySize(list[i])
		for size > b.left && len(list[i].Matches) > 0 {
			list[i].Matches = list[i].Matches[:len(list[i].Matches)/2]
			cut = true
			size = entrySize(list[i])
		}
		if cut {
			b.capped++
		}
		if size > b.left {
			b.dropped += len(list) - i
			return list[:i]
		}
		b.left -= size
	}
	return list
}

func (b *sizeBudget) notice() string {
	if b.dropped == 0 && b.capped == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n[partial result: %d of %d matching entries returned, alphabetical by path",
		b.total-b.dropped, b.total)
	if b.dropped > 0 {
		fmt.Fprintf(&sb, "; %d omitted to stay under %d KiB", b.dropped, maxFindResultBytes>>10)
	}
	if b.capped > 0 {
		fmt.Fprintf(&sb, "; matches cut short in %d of them", b.capped)
	}
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
		Description: `Locate files: what a directory holds (list), which paths match a name pattern (glob), which files contain a string (search, grep by RE2 regex).
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
