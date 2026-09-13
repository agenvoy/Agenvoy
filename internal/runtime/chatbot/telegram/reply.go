package telegram

import (
	"context"
	"html"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	go_bot_telegram "github.com/pardnchiu/go-bot/telegram"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot"
)

var tsPrefixRegex = regexp.MustCompile(`^ts:\d+\n`)

func (b *Bot) newReply(ctx context.Context, chatID int64, chatName, sessionID, replyTo string) chatbot.Reply {
	return chatbot.Reply{
		Channel:   chatbot.Telegram,
		SessionID: sessionID,
		ReplyTo:   replyTo,
		Clean: func(text string) string {
			return sanitizeHTML(tsPrefixRegex.ReplaceAllString(text, ""))
		},
		Status: func(text string) {
			id, _ := strconv.Atoi(replyTo)
			wrapped := "<blockquote expandable>" + html.EscapeString(text) + "</blockquote>"
			if err := b.client.SendStatus(ctx, chatID, id, wrapped, go_bot_telegram.WithStatusSendType(go_bot_telegram.TypeHTML)); err != nil {
				slog.Debug("github.com/pardnchiu/go-bot/telegram Bot.client.SendStatus",
					slog.String("session", sessionID),
					slog.String("chat", chatName),
					slog.String("text", text),
					slog.String("error", err.Error()))
			}
		},
		Finish: func() {
			if err := b.client.FinishStatus(ctx, chatID); err != nil {
				slog.Debug("github.com/pardnchiu/go-bot/telegram Bot.client.FinishStatus",
					slog.String("session", sessionID),
					slog.String("chat", chatName),
					slog.String("error", err.Error()))
			}
		},
		Send: func(replyTo, text string) (string, error) {
			id, _ := strconv.Atoi(replyTo)
			msg, err := b.client.Send(ctx, chatID, id, text, go_bot_telegram.WithSendType(go_bot_telegram.TypeHTML))
			if err != nil {
				return "", err
			}
			if msg == nil {
				return "", nil
			}
			return strconv.Itoa(msg.ID), nil
		},
		Attach: func(_ string, paths []string) {
			var photoPaths, docPaths []string
			for _, p := range paths {
				if imageExts[strings.ToLower(filepath.Ext(p))] {
					photoPaths = append(photoPaths, p)
					continue
				}
				docPaths = append(docPaths, p)
			}
			sendAttachments(context.WithoutCancel(ctx), chatID, chatName, photoPaths, docPaths)
		},
	}
}
