package historyStore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	ActionCreate     = "create"
	ActionModify     = "modify"
	ActionDelete     = "delete"
	maxSnapshotBytes = 1 << 20
)

type Meta struct {
	SessionID string
	TaskID    string
	Tool      string
}

type Change struct {
	dir       string
	name      string
	action    string
	content   []byte
	hash      string
	size      int64
	truncated bool
	trashPath string
}

func Capture(path string) (Change, error) {
	if conn == nil || path == "" {
		return Change{}, nil
	}

	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return Change{dir: filepath.Dir(path), name: filepath.Base(path), action: ActionCreate}, nil
	case err != nil:
		return Change{}, fmt.Errorf("os.Stat [%s]: %w", path, err)
	case info.IsDir():
		return Change{}, nil
	case info.Size() > maxSnapshotBytes:
		return oversized(path, info.Size()), nil
	}

	content, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return Change{}, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem ReadText [%s]: %w", path, err)
	}
	return CaptureContent(path, content), nil
}

func oversized(path string, size int64) Change {
	c := Change{
		dir:    filepath.Dir(path),
		name:   filepath.Base(path),
		action: ActionModify,
		size:   size,
	}

	tempPath, err := filesystem.CopyToStoreTemp(path)
	if err != nil {
		slog.Warn("filesystem.CopyToStoreTemp",
			slog.String("path", path),
			slog.String("error", err.Error()))
		c.truncated = true
		return c
	}
	c.trashPath = tempPath
	return c
}

func CaptureContent(path, content string) Change {
	if conn == nil || path == "" {
		return Change{}
	}
	return withContent(Change{
		dir:    filepath.Dir(path),
		name:   filepath.Base(path),
		action: ActionModify,
	}, content)
}

func (c Change) WithCreated(content string) Change {
	if c.action != ActionCreate {
		return c
	}
	return withContent(c, content)
}

func withContent(c Change, content string) Change {
	c.size = int64(len(content))
	if len(content) > maxSnapshotBytes {
		c.truncated = true
		return c
	}

	sum := sha256.Sum256([]byte(content))
	c.content = []byte(content)
	c.hash = hex.EncodeToString(sum[:])
	return c
}

func Record(ctx context.Context, c Change, meta Meta) error {
	if conn == nil || c.action == "" {
		return nil
	}
	if c.action == ActionModify && c.hash != "" && c.hash == latestModifyHash(ctx, c.dir, c.name) {
		return nil
	}
	return insert(ctx, c, meta)
}

func RecordDelete(ctx context.Context, path, trashPath string, meta Meta) error {
	if conn == nil || path == "" {
		return nil
	}

	c := Change{
		dir:       filepath.Dir(path),
		name:      filepath.Base(path),
		action:    ActionDelete,
		trashPath: trashPath,
	}
	if info, err := os.Stat(trashPath); err == nil && !info.IsDir() {
		c.size = info.Size()
	}
	return insert(ctx, c, meta)
}
