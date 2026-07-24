package exec

import (
	"context"
	"sync"
)

var (
	cancelMu  sync.Mutex
	cancelFns = map[string]context.CancelFunc{}
)

func registerCancel(taskID string, cancel context.CancelFunc) {
	if taskID == "" {
		return
	}
	cancelMu.Lock()
	cancelFns[taskID] = cancel
	cancelMu.Unlock()
}

func unregisterCancel(taskID string) {
	cancelMu.Lock()
	delete(cancelFns, taskID)
	cancelMu.Unlock()
}

func CancelTask(taskID string) bool {
	cancelMu.Lock()
	cancel, ok := cancelFns[taskID]
	cancelMu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}
