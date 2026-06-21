package tool

import (
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/runtime/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/telegram"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func Register() {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return
	}

	tgOK := cfg.TelegramEnabled && keychain.Get(telegram.Key) != ""
	dcOK := cfg.DiscordEnabled && keychain.Get(discord.Key) != ""
	if !tgOK && !dcOK {
		return
	}

	platformEnum = nil
	if tgOK {
		platformEnum = append(platformEnum, platformTelegram)
	}
	if dcOK {
		platformEnum = append(platformEnum, platformDiscord)
	}

	registChatbotFormat()
	registListChatbot()
	registSendToChatbot()
}
