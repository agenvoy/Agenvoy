package fast

import (
	"context"
	"sync/atomic"

	provider "github.com/pardnchiu/go-llm-router/core"
)

var (
	enable atomic.Bool
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

type ctxKey struct{}

func With(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, ctxKey{}, enabled)
}

func Mode(ctx context.Context) provider.Mode {
	enabled, ok := ctx.Value(ctxKey{}).(bool)
	if !ok {
		enabled = enable.Load()
	}
	if enabled {
		return provider.ModeFast
	}
	return provider.ModeDefault
}
