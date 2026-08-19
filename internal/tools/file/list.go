package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func listBatch(ctx context.Context, e *toolTypes.Executor, queries []findQuery) (string, error) {
	out := make(map[string]any, len(queries))
	for _, q := range queries {
		files, err := listOne(ctx, e, q.Dir, q.Recursive)
		if err != nil {
			out[q.Dir] = "error: " + err.Error()
			continue
		}
		out[q.Dir] = files
		if err := ctx.Err(); err != nil {
			return "", err
		}
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
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
