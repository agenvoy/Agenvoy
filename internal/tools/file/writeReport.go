package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

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
		Description: `Saves one long-form deliverable as a .md file and returns the write receipt with its path.
Use for the research / analysis / comparison / report body the reply summarises instead of reprinting.
Any other file, or a change to a file that already exists → edit_file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The complete report as markdown, not a diff.",
				},
			},
			"required": []string{"content"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			if err := rejectElided(params.Content, nil); err != nil {
				return "", err
			}

			return writeFileContent(ctx, e, reportPath(), params.Content, "write_report")
		},
	})
}

func reportPath() string {
	base := filesystem.DownloadDir
	if home, err := os.UserHomeDir(); err == nil && go_pkg_filesystem_reader.IsDir(filepath.Join(home, "Downloads")) {
		base = filepath.Join(home, "Downloads")
	}
	return filepath.Join(base, "report-"+time.Now().Format("20060102-150405")+".md")
}
