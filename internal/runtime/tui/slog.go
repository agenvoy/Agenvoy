package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

type Log struct {
	Source string
	Level  string
	Time   time.Time
	Msg    string
	Attrs  []slog.Attr
}

type tuiSlogHandler struct{}

func (h *tuiSlogHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= slog.LevelInfo
}

func (h *tuiSlogHandler) Handle(_ context.Context, r slog.Record) error {
	if isNoisySlog(r) {
		return nil
	}
	entry := Log{
		Source: "tui",
		Level:  levelLabel(r.Level),
		Time:   r.Time,
		Msg:    r.Message,
	}
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs = append(entry.Attrs, a)
		return true
	})
	go send(entry)
	return nil
}

func isNoisySlog(r slog.Record) bool {
	noisy := false
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" && strings.Contains(fmt.Sprintf("%v", a.Value.Any()), "unexpected end of JSON input") {
			noisy = true
			return false
		}
		return true
	})
	return noisy
}

func (h *tuiSlogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *tuiSlogHandler) WithGroup(_ string) slog.Handler      { return h }

func levelLabel(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

func levelLineStyle(l string) lipgloss.Style {
	switch l {
	case "ERROR":
		return errorStyle
	case "WARN":
		return warnStyle
	}
	return hintStyle
}

func renderLogLine(e Log) string {
	body := e.Msg
	for _, a := range e.Attrs {
		body += " " + a.Key + "=" + fmt.Sprintf("%v", a.Value.Any())
	}
	body = strings.TrimSpace(body)
	if strings.HasPrefix(e.Msg, "Telegram Verification Code") {
		code := extractField(body, "code=")
		username := extractField(body, "name=")
		line := fmt.Sprintf("$ Telegram Verification Code: %s (%s)", code, username)
		return systemStyle.Render(line) + "\n"
	}
	if strings.HasPrefix(e.Msg, "⎯ host reloaded") {
		line := "$ [" + e.Source + "] " + body + " - " + e.Time.Format("15:04:05")
		return warnStyle.Render(line) + "\n"
	}
	if strings.HasPrefix(e.Msg, "Discord Verification Code") {
		code := extractField(body, "code=")
		username := extractField(body, "name=")
		line := fmt.Sprintf("$ Discord Verification Code: %s (%s)", code, username)
		return systemStyle.Render(line) + "\n"
	}
	line := "$ [" + e.Source + "] " + body + " - " + e.Time.Format("15:04:05")
	return levelLineStyle(e.Level).Render(line) + "\n"
}

func extractField(s, key string) string {
	_, rest, ok := strings.Cut(s, key)
	if !ok {
		return ""
	}
	val, _, _ := strings.Cut(rest, " ")
	return val
}

type sessionSubMgr struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	current string
}

var subMgr = &sessionSubMgr{}

func installSlogTUI(ctx context.Context) func() {
	prev := slog.Default()
	slog.SetDefault(slog.New(&tuiSlogHandler{}))
	subMgrParentCtx = ctx
	daemonCtx, daemonCancel := context.WithCancel(ctx)
	go subscribeSessionEvents(daemonCtx, "daemon")

	return func() {
		daemonCancel()
		subMgr.Stop()
		slog.SetDefault(prev)
	}
}

var subMgrParentCtx = context.Background()

// * unsubscribe self tui logs
func subscribeSessionLog(sessionID string) {
	_ = sessionID
	subMgr.Switch(subMgrParentCtx, "")
}

func (m *sessionSubMgr) Switch(parent context.Context, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sessionID == m.current && m.cancel != nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.current = sessionID
	if sessionID == "" {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	go subscribeSessionEvents(ctx, sessionID)
}

func (m *sessionSubMgr) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.current = ""
}

func subscribeSessionEvents(ctx context.Context, sessionID string) {
	daemon.Stream(ctx, "/v1/session/"+sessionID+"/log", func(raw []byte) {
		var ev agentTypes.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			return
		}
		if ev.Type == agentTypes.EventDaemonLog {
			send(Log{
				Source: "daemon",
				Level:  ev.Source,
				Time:   time.Now(),
				Msg:    ev.Text,
			})
			return
		}
		send(agentEvent{event: ev})
	})
}
