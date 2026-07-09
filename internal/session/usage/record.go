package usage

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

const (
	maxLogSize     = 1 << 20
	trimTargetSize = 768 << 10
)

var mu sync.Mutex

func Append(sessionID, providerName, model, reasoning string, u agentTypes.Usage) {
	if sessionID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sessionID)) {
		return
	}

	path := filesystem.UsageLogPath(sessionID)
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s][%s@%s] in/%-7d out/%-7d write/%-7d hit/%-7d", ts, providerName, model, u.Input, u.Output, u.CacheCreate, u.CacheRead)
	if reasoning != "" {
		line += fmt.Sprintf(" reasoning/%s", reasoning)
	}
	line += "\n"
	if err := go_pkg_filesystem.AppendText(path, line); err != nil {
		slog.Warn("AppendText",
			slog.String("file", path),
			slog.String("error", err.Error()))
		return
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxLogSize {
		return
	}
	trim(path)
}

func trim(path string) {
	text, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem ReadText",
			slog.String("file", path),
			slog.String("error", err.Error()))
		return
	}

	data := []byte(text)
	if int64(len(data)) <= maxLogSize {
		return
	}

	cut := max(len(data)-trimTargetSize, 0)
	for cut < len(data) && data[cut] != '\n' {
		cut++
	}
	if cut < len(data) {
		cut++
	}
	if cut >= len(data) {
		if err := go_pkg_filesystem.WriteFile(path, "", 0644); err != nil {
			slog.Warn("github.com/pardnchiu/go-pkg/filesystem WriteFile",
				slog.String("file", path),
				slog.String("error", err.Error()))
		}
		return
	}
	if err := go_pkg_filesystem.WriteFile(path, string(data[cut:]), 0644); err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem WriteFile",
			slog.String("file", path),
			slog.String("error", err.Error()))
	}
}
