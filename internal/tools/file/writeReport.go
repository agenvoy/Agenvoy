package file

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registWriteReport() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "write_report",
		SystemUse:   true,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  false,
		Description: `Saves one long-form deliverable as a .md file under the work directory and returns the write receipt.
Use for the research / analysis / comparison / report body the reply summarises instead of reprinting.
Any other file, or a change to a file that already exists → edit_file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File name under the work directory, e.g. 'tsmc-2026-09-06.md'. Relative only; it never escapes that directory.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The complete report as markdown, not a diff.",
				},
			},
			"required": []string{"path", "content"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			if err := rejectElided(params.Content, nil); err != nil {
				return "", err
			}

			absPath, err := reportPath(e, params.Path)
			if err != nil {
				return "", err
			}
			return writeFileContent(ctx, e, absPath, params.Content, "write_report")
		},
	})
}

func reportPath(e *toolTypes.Executor, path string) (string, error) {
	base := e.WorkDir
	if base == "" {
		base = filesystem.DownloadDir
	}
	base = filepath.Clean(base)

	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	absPath := filepath.Clean(filepath.Join(base, path))
	if !strings.HasPrefix(absPath, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path must stay within %s", base)
	}
	return absPath, nil
}
