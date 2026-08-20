package file

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var sensitiveFiles = []string{
	"id_rsa", "id_rsa.pub",
	"id_dsa", "id_dsa.pub",
	"id_ecdsa", "id_ecdsa.pub",
	"id_ed25519", "id_ed25519.pub",
	"authorized_keys",
	"ssh_host_rsa_key", "ssh_host_rsa_key.pub",
	"ssh_host_ed25519_key", "ssh_host_ed25519_key.pub",
	".netrc", ".git-credentials", ".env",
}

var sensitiveExts = []string{".pem", ".key", ".p12", ".pfx", ".cer"}

func IsSensitivePath(absPath string) bool {
	base := filepath.Base(absPath)
	if slices.Contains(sensitiveFiles, base) || strings.HasPrefix(base, ".env.") {
		return true
	}
	return slices.Contains(sensitiveExts, strings.ToLower(filepath.Ext(base)))
}

const (
	maxReadSize      = 1 << 20
	defaultReadLimit = 1 << 30
)

func registReadFiles() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "read_files",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `Canonical way to read any file — text, PDF, DOCX, PPTX, CSV/TSV or image — and the step that must precede edit_file(mode=patch) unless it was already read this session.
Use for 讀檔 / 看一下這個檔案 / 這份 PDF 寫什麼, and for read_file / cat / head / tail.
Each path maps to its content, or to an error string for that path. Locating a file → find_files; opening it in an app → open_file.`,
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

			result, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(result), nil
		},
	})
}

func readOne(ctx context.Context, e *toolTypes.Executor, baseDir, path string, offset, limit int) (string, error) {
	absPath, err := go_pkg_filesystem.AbsPath(baseDir, path, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: AbsPath: %w", err)
	}
	if absPath == "" {
		return "", fmt.Errorf("path is required")
	}

	if parent, ok := denied.Hit(e.SessionID, absPath); ok {
		return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
	}

	offset = max(offset, 1)
	limit = max(limit, 0)
	if limit == 0 {
		limit = defaultReadLimit
	}
	out, err := filesystem.ReadFile(ctx, absPath, offset, limit)
	if err != nil && denied.IsPermission(err) {
		denied.Register(e.SessionID, absPath)
		return "", fmt.Errorf("permission denied: %s (recorded; further reads under this path will be skipped)", absPath)
	}
	return out, err
}
