package configStatus

import (
	"log/slog"
	"sync"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	StatusOnline = "online"
	StatusIdle   = "idle"
)

var (
	mu sync.Mutex
)

type Status struct {
	State string `json:"state"`
	Count int    `json:"count"`
}

func Online(sessionID string) {
	if sessionID == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	status := get(sessionID)
	status.Count++
	status.State = StatusOnline
	write(sessionID, status)
}

func Idle(sessionID string) {
	if sessionID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	status := get(sessionID)
	status.Count--
	if status.Count <= 0 {
		status.Count = 0
		status.State = StatusIdle
	} else {
		status.State = StatusOnline
	}
	write(sessionID, status)
}

func get(sessionID string) Status {
	status, err := go_pkg_filesystem.ReadJSON[Status](filesystem.StatusPath(sessionID))
	if err != nil {
		return Status{State: StatusIdle}
	}
	if status.Count < 0 {
		status.Count = 0
	}
	if status.State == "" {
		if status.Count > 0 {
			status.State = StatusOnline
		} else {
			status.State = StatusIdle
		}
	}
	return status
}

func write(sessionID string, status Status) {
	if err := go_pkg_filesystem.WriteJSON(filesystem.StatusPath(sessionID), status, true); err != nil {
		slog.Debug("github.com/pardnchiu/go-pkg/filesystem WriteJSON",
			slog.String("file", filesystem.StatusPath(sessionID)),
			slog.String("error", err.Error()))
	}
}
