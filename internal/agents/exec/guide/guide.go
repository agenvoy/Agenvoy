package guide

import (
	"context"
	"path/filepath"
	"slices"
	"sync/atomic"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

var (
	enable atomic.Bool
	Files  = []string{"CLAUDE.md", "AGENTS.md"}
)

func Enable() {
	enable.Store(true)
}

func Disable() {
	enable.Store(false)
}

func IsEnabled() bool {
	return enable.Load()
}

func IsEnabledCtx(ctx context.Context) bool {
	if enabled, ok := ctx.Value(ctxKey{}).(bool); ok {
		return enabled
	}
	return enable.Load()
}

type ctxKey struct{}

func With(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, ctxKey{}, enabled)
}

func Name(workDir string) string {
	if i := slices.IndexFunc(Files, func(name string) bool {
		return go_pkg_filesystem_reader.IsFile(filepath.Join(workDir, name))
	}); i >= 0 {
		return Files[i]
	}
	return ""
}
