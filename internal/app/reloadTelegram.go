package app

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/telegram"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

var (
	telegramMu          sync.Mutex
	telegramBot         *telegram.Bot
	lastTelegramEnabled bool
	lastTelegramToken   string
)

func ReloadTelegram(attempt int) {
	newToken := keychain.Get(telegram.Key)
	newEnabled := false
	if cfg, err := config.Load(); err == nil && cfg != nil {
		newEnabled = cfg.TelegramEnabled
	}

	telegramMu.Lock()
	defer telegramMu.Unlock()

	if attempt == 0 && newEnabled == lastTelegramEnabled && newToken == lastTelegramToken {
		return
	}

	if telegramBot != nil {
		_ = telegram.Close(telegramBot)
		telegramBot = nil
	}

	if !newEnabled || newToken == "" {
		lastTelegramEnabled = newEnabled
		lastTelegramToken = newToken
		return
	}

	bot, err := telegram.New()
	if err != nil {
		slog.Error("telegram.New",
			slog.String("error", err.Error()),
			slog.Int("attempt", attempt))
		if attempt < reloadRetryMax {
			go func() {
				time.Sleep(reloadRetryDelay)
				ReloadTelegram(attempt + 1)
			}()
		}
		return
	}
	lastTelegramEnabled = newEnabled
	lastTelegramToken = newToken
	telegramBot = bot
}

func CloseTelegram() {
	telegramMu.Lock()
	if telegramBot != nil {
		_ = telegram.Close(telegramBot)
		telegramBot = nil
	}
	telegramMu.Unlock()
}
