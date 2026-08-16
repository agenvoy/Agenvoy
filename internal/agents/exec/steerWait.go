package exec

import (
	"fmt"
	"strings"
	"sync"

	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
)

var (
	steerMap   sync.Map
	runningMap sync.Map
	runningMu  sync.Mutex
)

type steerEntry struct {
	mu   sync.Mutex
	list []string
}

type runningEntry struct {
	count int
}

func markRunning(sessionID string) func() {
	if sessionID == "" {
		return func() {}
	}
	v, _ := runningMap.LoadOrStore(sessionID, &runningEntry{})
	e := v.(*runningEntry)
	runningMu.Lock()
	e.count++
	runningMu.Unlock()

	return func() {
		runningMu.Lock()
		e.count--
		done := e.count <= 0
		runningMu.Unlock()
		if done {
			runningMap.Delete(sessionID)
		}
	}
}

func IsRunning(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	v, ok := runningMap.Load(sessionID)
	if !ok {
		return false
	}
	runningMu.Lock()
	defer runningMu.Unlock()
	return v.(*runningEntry).count > 0
}

func AppendSteer(sessionID, text string) {
	v, _ := steerMap.LoadOrStore(sessionID, &steerEntry{})
	e := v.(*steerEntry)
	e.mu.Lock()
	e.list = append(e.list, text)
	e.mu.Unlock()
	sessionLog.Steer(sessionID, text)
}

func getSteer(sessionID string) []string {
	v, ok := steerMap.Load(sessionID)
	if !ok {
		return nil
	}
	e := v.(*steerEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.list) == 0 {
		return nil
	}
	out := e.list
	e.list = nil
	return out
}

func ClearSteer(sessionID string) {
	steerMap.Delete(sessionID)
}

func formatSteerInjection(pending []string) string {
	return fmt.Sprintf("[使用者在任務進行中插話，任務尚未結束。請評估這則插話是否改變當前計畫的方向、範疇或優先序，並據此決定繼續原計畫、調整計畫，或依插話內容行動]\n%s", strings.Join(pending, "\n"))
}
