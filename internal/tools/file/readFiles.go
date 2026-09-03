package file

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	maxReadSize      = 1 << 20
	defaultReadLimit = 1 << 30
)

func registReadFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "read_files",
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Canonical way to read any file — text, PDF, DOCX, PPTX, CSV/TSV or image — and the step that must precede edit_file(mode=patch) unless it was already read this session.
Use for 讀檔 / 看一下這個檔案 / 這份 PDF 寫什麼, and for read_file / cat / head / tail.
Each path maps to its content, or to an error string for that path. Text lines arrive as "<row>\t<line>" — the number is not in the file, so strip it before using a line as an edit_file anchor. Locating a file → find_files; opening it in an app → open_file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"files": map[string]any{
					"type":        "array",
					"description": "Every file in one call rather than repeated calls.",
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
								"description": "How many lines (pages, slides, rows) to read. Omit to read the whole file; set it only to page through one that hits the 1MB cap.",
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
					Path   string `json:"path"`
					Offset int    `json:"offset"`
					Limit  int    `json:"limit"`
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
				content, err := readOne(ctx, e, baseDir, f.Path, f.Offset, f.Limit)
				if err != nil {
					content = "error: " + err.Error()
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

func readOne(ctx context.Context, e *toolTypes.Executor, baseDir, path string, offset, limit int) (string, error) {
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
	return filesystem.ReadFile(ctx, absPath, offset, limit)
}
