package app

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/line"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

var (
	lineMu          sync.Mutex
	lineBot         *line.Bot
	lastLineEnabled bool
	lastLineSecret  string
	lastLineToken   string
)

func ReloadLine(attempt int) {
	newSecret := keychain.Get(line.SecretKey)
	newToken := keychain.Get(line.TokenKey)
	newEnabled := false
	if cfg, err := config.Load(); err == nil && cfg != nil {
		newEnabled = cfg.LineEnabled
	}

	lineMu.Lock()
	defer lineMu.Unlock()

	if attempt == 0 && newEnabled == lastLineEnabled && newSecret == lastLineSecret && newToken == lastLineToken {
		return
	}

	if lineBot != nil {
		_ = line.Close(lineBot)
		lineBot = nil
	}

	if !newEnabled || newSecret == "" || newToken == "" {
		lastLineEnabled = newEnabled
		lastLineSecret = newSecret
		lastLineToken = newToken
		return
	}

	bot, err := line.New()
	if err != nil {
		slog.Error("line.New",
			slog.String("error", err.Error()),
			slog.Int("attempt", attempt))
		if attempt < reloadRetryMax {
			go func() {
				time.Sleep(reloadRetryDelay)
				ReloadLine(attempt + 1)
			}()
		}
		return
	}
	lastLineEnabled = newEnabled
	lastLineSecret = newSecret
	lastLineToken = newToken
	lineBot = bot
}

func CloseLine() {
	lineMu.Lock()
	if lineBot != nil {
		_ = line.Close(lineBot)
		lineBot = nil
	}
	lineMu.Unlock()
}
