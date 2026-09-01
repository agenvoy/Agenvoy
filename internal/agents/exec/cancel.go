package exec

import (
	"context"
	"sync"
)

var (
	cancelMu  sync.Mutex
	cancelFns = map[string]context.CancelFunc{}
)

func registerCancel(onceID string, cancel context.CancelFunc) {
	if onceID == "" {
		return
	}
	cancelMu.Lock()
	cancelFns[onceID] = cancel
	cancelMu.Unlock()
}

func unregisterCancel(onceID string) {
	cancelMu.Lock()
	delete(cancelFns, onceID)
	cancelMu.Unlock()
}

func CancelTask(onceID string) bool {
	cancelMu.Lock()
	cancel, ok := cancelFns[onceID]
	cancelMu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}
