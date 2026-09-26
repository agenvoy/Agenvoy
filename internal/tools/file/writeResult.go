package file

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registWriteResult() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "write_result",
		SystemUse:   true,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  false,
		Description: `Saves one long-form deliverable as a .md or .html file and returns the write receipt with its path.
Use for the research / analysis / comparison / report body the reply summarises instead of reprinting, and for a finished HTML page.
Any other file, or a change to a file that already exists → edit_file.
Reach for it past roughly 400 words, more than two sections, or a table plus commentary; the same message still carries every key finding, figure and decision.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"md", "html"},
					"default":     "md",
					"description": "md: markdown report. html: one self-contained HTML page.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The complete file in the chosen format, not a diff.",
				},
			},
			"required": []string{"content"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Format  string `json:"format"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			switch params.Format {
			case "":
				params.Format = "md"
			case "md", "html":
			default:
				return "", fmt.Errorf("unknown format %q; available: md, html", params.Format)
			}

			if err := rejectElided(params.Content, nil); err != nil {
				return "", err
			}

			return writeFileContent(ctx, e, resultPath(params.Format), params.Content, "write_result")
		},
	})
}

func resultPath(ext string) string {
	return filepath.Join(filesystem.OutputDir(), "report-"+time.Now().Format("20060102-150405")+"."+ext)
}
