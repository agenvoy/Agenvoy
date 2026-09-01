package runtime

import "sync"

var (
	cancelMu       sync.RWMutex
	cancelNotifier func(sessionID, taskHash, reason string)
)

func RegisterCancelNotifier(fn func(sessionID, taskHash, reason string)) {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	cancelNotifier = fn
}

func NotifyCanceled(sessionID, taskHash, reason string) {
	cancelMu.RLock()
	fn := cancelNotifier
	cancelMu.RUnlock()
	if fn == nil || sessionID == "" {
		return
	}
	fn(sessionID, taskHash, reason)
}
