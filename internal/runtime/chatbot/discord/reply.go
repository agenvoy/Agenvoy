package discord

import (
	"context"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot"
)

func (b *Bot) newReply(ctx context.Context, channelID, channelName, sessionID, replyTo string) chatbot.Reply {
	return chatbot.Reply{
		Channel:   chatbot.Discord,
		SessionID: sessionID,
		ReplyTo:   replyTo,
		Status: func(text string) {
			if err := b.client.SendStatus(ctx, channelID, replyTo, text); err != nil {
				slog.Debug("github.com/pardnchiu/go-bot/discord Bot.client.SendStatus",
					slog.String("session", sessionID),
					slog.String("channel", channelName),
					slog.String("text", text),
					slog.String("error", err.Error()))
			}
		},
		Finish: func() {
			if err := b.client.FinishStatus(ctx, channelID); err != nil {
				slog.Debug("github.com/pardnchiu/go-bot/discord Bot.client.FinishStatus",
					slog.String("session", sessionID),
					slog.String("channel", channelName),
					slog.String("error", err.Error()))
			}
		},
		Send: func(replyTo, text string) (string, error) {
			msg, err := b.client.Send(ctx, channelID, replyTo, text)
			if err != nil {
				return "", err
			}
			if msg == nil {
				return "", nil
			}
			return msg.ID, nil
		},
		Attach: func(replyTo string, paths []string) {
			sendAttachments(context.WithoutCancel(ctx), b.client, channelID, channelName, replyTo, paths)
		},
	}
}
