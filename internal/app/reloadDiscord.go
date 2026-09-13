package app

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/discord"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

var (
	discordMu          sync.Mutex
	discordBot         *discord.Bot
	lastDiscordEnabled bool
	lastDiscordToken   string
)

func ReloadDiscord(attempt int) {
	newToken := keychain.Get(discord.Key)
	newEnabled := false
	if cfg, err := config.Load(); err == nil && cfg != nil {
		newEnabled = cfg.DiscordEnabled
	}

	discordMu.Lock()
	defer discordMu.Unlock()

	if attempt == 0 && newEnabled == lastDiscordEnabled && newToken == lastDiscordToken {
		return
	}

	if discordBot != nil {
		_ = discord.Close(discordBot)
		discordBot = nil
	}

	if !newEnabled || newToken == "" {
		lastDiscordEnabled = newEnabled
		lastDiscordToken = newToken
		return
	}

	bot, err := discord.New()
	if err != nil {
		slog.Error("discord.New",
			slog.String("error", err.Error()),
			slog.Int("attempt", attempt))
		if attempt < reloadRetryMax {
			go func() {
				time.Sleep(reloadRetryDelay)
				ReloadDiscord(attempt + 1)
			}()
		}
		return
	}
	lastDiscordEnabled = newEnabled
	lastDiscordToken = newToken
	discordBot = bot
}

func CloseDiscord() {
	discordMu.Lock()
	if discordBot != nil {
		_ = discord.Close(discordBot)
		discordBot = nil
	}
	discordMu.Unlock()
}
