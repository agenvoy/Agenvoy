package exec

import (
	"fmt"
	"strings"
	"sync"
)

var steerMap sync.Map

type steerEntry struct {
	mu   sync.Mutex
	list []string
}

func AppendSteer(sessionID, text string) {
	v, _ := steerMap.LoadOrStore(sessionID, &steerEntry{})
	e := v.(*steerEntry)
	e.mu.Lock()
	e.list = append(e.list, text)
	e.mu.Unlock()
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
