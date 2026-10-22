package execChatbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type SendFn func(ctx context.Context, targetID, str string) error

var (
	adminMu  sync.RWMutex
	adminMap = map[string]SendFn{}
)

func RegisterAdmin(chatbot string, fn SendFn) {
	adminMu.Lock()
	adminMap[chatbot] = fn
	adminMu.Unlock()
}

func ParseAdmin(channel string) (chatbot, id string, ok bool) {
	channel = strings.TrimSpace(channel)
	switch {
	case strings.HasPrefix(channel, "tg@"):
		chatbot, id = "tg", strings.TrimSpace(channel[len("tg@"):])
	case strings.HasPrefix(channel, "dc@"):
		chatbot, id = "dc", strings.TrimSpace(channel[len("dc@"):])
	case strings.HasPrefix(channel, "ln@"):
		chatbot, id = "ln", strings.TrimSpace(channel[len("ln@"):])
	default:
		return "", "", false
	}
	if id == "" {
		return "", "", false
	}
	return chatbot, id, true
}

func NotifyAdmin(ctx context.Context, code, sourceName string) {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return
	}

	channel := strings.TrimSpace(cfg.AdminChannel)
	if channel == "" {
		return
	}

	chatbot, id, ok := ParseAdmin(channel)
	if !ok {
		slog.Warn("admin_channel parse failed; verification code stays log-only",
			slog.String("value", channel))
		return
	}

	var authPath string
	switch chatbot {
	case "tg":
		authPath = filesystem.TelegramAuthPath
	case "dc":
		authPath = filesystem.DiscordAuthPath
	case "ln":
		authPath = filesystem.LineAuthPath
	}
	if !utils.IsAuthorized(authPath, id) {
		slog.Warn("admin_channel not verified; verification code stays log-only",
			slog.String("channel", channel))
		return
	}

	adminMu.RLock()
	fn, ok := adminMap[chatbot]
	adminMu.RUnlock()
	if !ok {
		return
	}

	str := fmt.Sprintf("%s requested access · verification code: %s", sourceName, code)
	if err := fn(ctx, id, str); err != nil {
		slog.Warn("admin_channel send failed",
			slog.String("channel", channel),
			slog.String("error", err.Error()))
	}
}
