package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem/record"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
)

const (
	daemonLogChannel = "daemon"
)

type daemonSlogHandler struct {
	base slog.Handler
}

func (h *daemonSlogHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.base.Enabled(ctx, l)
}

func (h *daemonSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelInfo {
		return h.base.Handle(ctx, r)
	}

	var sb strings.Builder
	sb.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteByte(' ')
		sb.WriteString(a.Key)
		sb.WriteByte('=')
		sb.WriteString(fmt.Sprintf("%v", a.Value.Any()))
		return true
	})

	select {
	case daemonLogQueue <- agentTypes.Event{
		Type:   agentTypes.EventDaemonLog,
		Source: r.Level.String(),
		Text:   strings.TrimSpace(sb.String()),
	}:
	default:
	}
	return h.base.Handle(ctx, r)
}

func (h *daemonSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &daemonSlogHandler{base: h.base.WithAttrs(attrs)}
}

func (h *daemonSlogHandler) WithGroup(name string) slog.Handler {
	return &daemonSlogHandler{base: h.base.WithGroup(name)}
}

var (
	daemonLogQueue = make(chan agentTypes.Event, 1024)
	daemonLogOnce  sync.Once
)

func installDaemonSlog() {
	daemonLogOnce.Do(func() {
		go func() {
			for event := range daemonLogQueue {
				pubsub.Pub(daemonLogChannel, event)
			}
		}()
	})

	writer := &daemonLogWriter{}
	if stat, err := os.Stderr.Stat(); err == nil {
		writer.size = stat.Size()
	}

	base := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(&daemonSlogHandler{base: base}))
}

type daemonLogWriter struct {
	mu   sync.Mutex
	size int64
}

func (w *daemonLogWriter) Write(raw []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := os.Stderr.Write(raw)
	w.size += int64(n)
	if w.size <= record.MaxLogSize {
		return n, err
	}
	if record.TrimLog() == nil {
		if stat, statErr := os.Stderr.Stat(); statErr == nil {
			w.size = stat.Size()
		}
	}
	return n, err
}
