package usage

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

const (
	maxLogSize     = 5 << 20
	trimTargetSize = 4 << 20
	retainDays     = 28
)

var mu sync.Mutex

func Append(sessionID, providerName, model string, u provider.Usage) {
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
	line := fmt.Sprintf("[%s][%s@%s] in/%-7d out/%-7d write/%-7d hit/%-7d\n", ts, providerName, model, u.Input, u.Output, u.CacheCreate, u.CacheRead)
	if err := go_pkg_filesystem.AppendText(path, line); err != nil {
		slog.Debug("AppendText",
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
		slog.Debug("github.com/pardnchiu/go-pkg/filesystem ReadText",
			slog.String("file", path),
			slog.String("error", err.Error()))
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retainDays)
	var kept strings.Builder
	for line := range strings.SplitSeq(text, "\n") {
		if line == "" {
			continue
		}
		matches := linePattern.FindStringSubmatch(line)
		if len(matches) != 7 {
			continue
		}
		at, parseErr := time.ParseInLocation(timestampLayout, matches[1], cutoff.Location())
		if parseErr != nil || at.Before(cutoff) {
			continue
		}
		kept.WriteString(line)
		kept.WriteByte('\n')
	}

	out := kept.String()
	if len(out) > maxLogSize {
		cut := len(out) - trimTargetSize
		for cut < len(out) && out[cut] != '\n' {
			cut++
		}
		if cut < len(out) {
			cut++
		}
		out = out[cut:]
	}

	if err := go_pkg_filesystem.WriteFile(path, out, 0644); err != nil {
		slog.Debug("github.com/pardnchiu/go-pkg/filesystem WriteFile",
			slog.String("file", path),
			slog.String("error", err.Error()))
	}
}
