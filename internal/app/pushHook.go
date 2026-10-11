package app

import (
	"context"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	execChatbot "github.com/pardnchiu/agenvoy/internal/agents/exec/chatbot"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/telegram"
)

func init() {
	exec.RegisterPushHook("dc-", discord.PushDiscordResult)
	exec.RegisterPushHook("tg-", telegram.PushTelegramResult)
	execChatbot.RegisterAdmin("tg", func(ctx context.Context, id, str string) error {
		return chatbot.SendAdminCode(ctx, chatbot.Telegram, id, str)
	})
	execChatbot.RegisterAdmin("dc", func(ctx context.Context, id, str string) error {
		return chatbot.SendAdminCode(ctx, chatbot.Discord, id, str)
	})
}
