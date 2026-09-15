package exec

import (
	"context"
	"sync"

	"github.com/pardnchiu/agenvoy/internal/runtime"
)

var (
	cancelMu  sync.Mutex
	cancelFns = map[string]context.CancelCauseFunc{}
)

func registerCancel(taskHash string, cancel context.CancelCauseFunc) {
	if taskHash == "" {
		return
	}
	cancelMu.Lock()
	cancelFns[taskHash] = cancel
	cancelMu.Unlock()
}

func unregisterCancel(taskHash string) {
	cancelMu.Lock()
	delete(cancelFns, taskHash)
	cancelMu.Unlock()
}

func CancelTask(taskHash string) bool {
	cancelMu.Lock()
	cancel, ok := cancelFns[taskHash]
	cancelMu.Unlock()
	if !ok {
		return false
	}
	cancel(runtime.ErrUserCanceled)
	return true
}
