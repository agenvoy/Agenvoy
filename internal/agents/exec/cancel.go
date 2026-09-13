package exec

import (
	"context"
	"sync"
)

var (
	cancelMu  sync.Mutex
	cancelFns = map[string]context.CancelFunc{}
)

func registerCancel(taskHash string, cancel context.CancelFunc) {
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
	cancel()
	return true
}
