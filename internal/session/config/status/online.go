package configStatus

import (
	"log/slog"
	"os"
	"sync"
	"syscall"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	StatusOnline = "online"
	StatusIdle   = "idle"
)

var (
	mu sync.Mutex
)

type Task struct {
	ID        string `json:"id"`
	Input     string `json:"input"`
	StartedAt string `json:"started_at"`
	PID       int    `json:"pid,omitempty"`
}

type Status struct {
	State   string `json:"state"`
	Active  []Task `json:"active"`
	EndedAt string `json:"ended_at"`
}

func Online(sessionID, input string) string {
	if sessionID == "" {
		return ""
	}

	mu.Lock()
	defer mu.Unlock()

	status := get(sessionID)
	task := Task{
		ID:        go_pkg_utils.UUID(),
		Input:     go_pkg_utils.TruncateString(input, 256),
		StartedAt: time.Now().Format("2006-01-02 15:04:05.000"),
		PID:       os.Getpid(),
	}
	status.Active = append(status.Active, task)
	status.State = StatusOnline
	write(sessionID, status)
	return task.ID
}

func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func get(sessionID string) Status {
	status, err := go_pkg_filesystem.ReadJSON[Status](filesystem.StatusPath(sessionID))
	if err != nil {
		return Status{State: StatusIdle}
	}

	live := status.Active[:0]
	stale := false
	for _, t := range status.Active {
		if isAlive(t.PID) {
			live = append(live, t)
		} else {
			stale = true
		}
	}
	status.Active = live

	if stale && len(status.Active) == 0 && status.EndedAt == "" {
		status.EndedAt = time.Now().Format("2006-01-02 15:04:05.000")
	}
	if status.State == "" || stale {
		if len(status.Active) > 0 {
			status.State = StatusOnline
		} else {
			status.State = StatusIdle
		}
	}
	if stale {
		write(sessionID, status)
	}
	return status
}

func write(sessionID string, status Status) {
	if status.Active == nil {
		status.Active = []Task{}
	}

	if err := go_pkg_filesystem.WriteJSON(filesystem.StatusPath(sessionID), status, true); err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem WriteJSON",
			slog.String("file", filesystem.StatusPath(sessionID)),
			slog.String("error", err.Error()))
	}
}

func Idle(sessionID, taskID string) {
	if sessionID == "" || taskID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	status := get(sessionID)
	filtered := status.Active[:0]
	for _, t := range status.Active {
		if t.ID == taskID {
			continue
		}
		filtered = append(filtered, t)
	}
	status.Active = filtered
	if len(status.Active) == 0 {
		status.State = StatusIdle
		status.EndedAt = time.Now().Format("2006-01-02 15:04:05.000")
	} else {
		status.State = StatusOnline
	}
	write(sessionID, status)
}
