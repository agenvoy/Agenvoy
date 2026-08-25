package configStatus

import (
	"log/slog"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func Get(sessionID string) Status {
	if sessionID == "" {
		return Status{}
	}
	mu.Lock()
	defer mu.Unlock()
	return get(sessionID)
}

func Reset() {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem/reader ListDirs",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}
	for _, dir := range dirs {
		reset(dir.Name)
	}
}

func reset(sessionID string) {
	if sessionID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	status, err := go_pkg_filesystem.ReadJSON[Status](filesystem.StatusPath(sessionID))
	if err != nil {
		return
	}
	if status.State == StatusIdle && status.Count == 0 {
		return
	}
	status.Count = 0
	status.State = StatusIdle
	write(sessionID, status)
}
