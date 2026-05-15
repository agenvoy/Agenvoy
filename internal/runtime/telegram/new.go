package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/go-bot/telegram"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

const Key = "TELEGRAM_TOKEN"

type Bot struct {
	client *telegram.Bot
	cancel context.CancelFunc
}

func New() (*Bot, error) {
	cfg, err := session.Load()
	if err != nil || cfg == nil || !cfg.TelegramEnabled {
		return nil, nil
	}
	token := keychain.Get(Key)
	if token == "" {
		return nil, nil
	}

	client, err := telegram.New(token)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-bot/telegram New: %w", err)
	}

	bot := &Bot{client: client}

	client.Reply(func(ctx context.Context, in telegram.Input) string {
		slog.Info("telegram message",
			slog.Int64("chat", in.ChatID),
			slog.String("from", in.Username),
			slog.String("text", in.Text))
		if err := run(ctx, bot, in); err != nil {
			slog.Warn("telegram run",
				slog.Int64("chat", in.ChatID),
				slog.String("error", err.Error()))
		}
		return ""
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := client.Start(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("github.com/pardnchiu/go-bot/telegram Start: %w", err)
	}
	bot.cancel = cancel

	slog.Info("telegram bot is running",
		slog.String("user", client.Status().Username))

	return bot, nil
}

func Close(b *Bot) error {
	if b == nil || b.client == nil {
		return nil
	}
	if b.cancel != nil {
		b.cancel()
	}
	return b.client.Close()
}
