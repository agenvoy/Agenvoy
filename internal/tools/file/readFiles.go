package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pardnchiu/agenvoy/internal/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	defaultReadLimit     = 1 << 11 // 2048
	defaultAroundContext = 20
)

func registReadFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "read_files",
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Canonical way to read any file — text, PDF, DOCX, PPTX, CSV/TSV, image, or audio/video (returned as a verbatim transcript) — and the step that must precede edit_file on an existing file: an edit is refused unless the file was read in this turn and is unchanged on disk since.
Use for 讀檔 / 看一下這個檔案 / 這份 PDF 寫什麼 / 這段錄音說了什麼, and for read_file / cat / head / tail.
Each path maps to its content, or to an error string for that path. Text lines arrive as "<row>\t<line>" — the number is not in the file, so strip it before using a line as an edit_file anchor. Locating a file → find_files; opening it in an app → open_file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"files": map[string]any{
					"type":        "array",
					"description": "Every file in one call rather than repeated calls. The same path may appear more than once; its results are joined in order.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"path": map[string]any{
								"type":        "string",
								"description": "File to read — '/abs/path/foo.go', '~/notes.md', 'relative/file.md'.",
							},
							"offset": map[string]any{
								"type":        "integer",
								"description": "1-based line — page for PDF, slide for PPTX, row for CSV.",
								"default":     1,
							},
							"limit": map[string]any{
								"type":        "integer",
								"description": "How many lines (pages, slides, rows) to read. When find_files(mode=search, output=content) already gave the line numbers, read only that region. A text file cut short ends with a notice naming the next offset.",
								"default":     defaultReadLimit,
							},
							"around": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "integer"},
								"description": "Plain-text files only: 1-based rows to read around, e.g. the lines find_files returned. Each row gets `context` lines on both sides, overlapping windows merge, and a '...' line marks skipped text. Set → offset and limit are ignored.",
							},
							"context": map[string]any{
								"type":        "integer",
								"description": "With around: lines shown before and after each row; 0 shows the rows alone.",
								"default":     defaultAroundContext,
							},
						},
						"required": []string{
							"path",
						},
					},
				},
			},
			"required": []string{
				"files",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Files []struct {
					Path    string `json:"path"`
					Offset  int    `json:"offset"`
					Limit   int    `json:"limit"`
					Around  []int  `json:"around"`
					Context *int   `json:"context"`
				} `json:"files"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Files) == 0 {
				return "", fmt.Errorf("files is required")
			}

			baseDir := e.WorkDir
			if baseDir == "" {
				baseDir = filesystem.DownloadDir
			}

			out := make(map[string]string, len(params.Files))
			for _, f := range params.Files {
				contextLines := defaultAroundContext
				if f.Context != nil {
					contextLines = *f.Context
				}
				content, err := readOne(ctx, e, baseDir, f.Path, f.Offset, f.Limit, f.Around, contextLines)
				if err != nil {
					content = "error: " + err.Error()
				}
				if prev, ok := out[f.Path]; ok {
					content = prev + "\n" + content
				}
				out[f.Path] = content
			}

			result, err := utils.MarshalPlain(out)
			if err != nil {
				return "", fmt.Errorf("utils.MarshalPlain: %w", err)
			}
			return string(result), nil
		},
	})
}

func readOne(ctx context.Context, e *toolTypes.Executor, baseDir, path string, offset, limit int, around []int, contextLines int) (string, error) {
	absPath, err := boundary.Resolve(e.SessionID, baseDir, path)
	if err != nil {
		return "", fmt.Errorf("boundary.Resolve: %w", err)
	}
	if absPath == "" {
		return "", fmt.Errorf("path is required")
	}

	offset = max(offset, 1)
	limit = max(limit, 0)
	if limit == 0 {
		limit = defaultReadLimit
	}

	info, statErr := os.Stat(absPath)
	var content string
	if len(around) > 0 {
		content, err = filesystem.ReadAround(absPath, around, contextLines)
	} else {
		content, err = filesystem.ReadFile(ctx, absPath, offset, limit)
	}
	if err != nil {
		return "", err
	}
	if statErr == nil {
		e.MarkRead(absPath, info.ModTime())
	}
	return content, nil
}
