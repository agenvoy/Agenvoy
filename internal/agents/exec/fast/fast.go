package fast

import (
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

func Mode() provider.Mode {
	if enable.Load() {
		return provider.ModeFast
	}
	return provider.ModeDefault
}
