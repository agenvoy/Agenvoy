package chatbot

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/utils"
	go_bot_discord "github.com/pardnchiu/go-bot/discord"
	go_bot_telegram "github.com/pardnchiu/go-bot/telegram"
	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

type Channel int

const (
	Telegram Channel = iota
	Discord
)

type SavedAttachment struct {
	Path       string
	Transcribe bool
}

func TranscribeSavedAttachments(ctx context.Context, attachments []SavedAttachment) ([]string, []string, error) {
	var transcripts []string
	var paths []string
	for _, attachment := range attachments {
		if attachment.Path == "" {
			continue
		}
		if !attachment.Transcribe {
			paths = append(paths, attachment.Path)
			continue
		}
		text, err := filesystem.TranscribeMedia(ctx, attachment.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("transcribe %s: %w", attachment.Path, err)
		}
		if text = strings.TrimSpace(text); text != "" {
			transcripts = append(transcripts, text)
		}
	}
	return transcripts, paths, nil
}

func SendAdminCode(ctx context.Context, ch Channel, targetID, text string) error {
	switch ch {
	case Telegram:
		token := strings.TrimSpace(keychain.Get("TELEGRAM_TOKEN"))
		if token == "" {
			return fmt.Errorf("telegram token missing")
		}
		id, err := strconv.ParseInt(strings.TrimSpace(targetID), 10, 64)
		if err != nil {
			return fmt.Errorf("parse chatID %q: %w", targetID, err)
		}
		client, err := go_bot_telegram.New(token)
		if err != nil {
			return fmt.Errorf("go-bot/telegram New: %w", err)
		}
		if _, err := client.Send(ctx, id, 0, html.EscapeString(text), go_bot_telegram.WithSendType(go_bot_telegram.TypeHTML)); err != nil {
			return fmt.Errorf("go-bot/telegram Send: %w", err)
		}
	case Discord:
		token := strings.TrimSpace(keychain.Get("DISCORD_TOKEN"))
		if token == "" {
			return fmt.Errorf("discord token missing")
		}
		client, err := go_bot_discord.New(token)
		if err != nil {
			return fmt.Errorf("go-bot/discord New: %w", err)
		}
		if _, err := client.Send(ctx, strings.TrimSpace(targetID), "", text); err != nil {
			return fmt.Errorf("go-bot/discord Send: %w", err)
		}
	default:
		return fmt.Errorf("unknown channel %d", ch)
	}
	return nil
}

func wrapBlock(ch Channel, text string) string {
	switch ch {
	case Telegram:
		return "<blockquote expandable>" + text + "</blockquote>"
	default:
		return "-# ⎿ " + text
	}
}

func BuildPushFooter(ctx context.Context, ch Channel, duration time.Duration, model string, usage *provider.Usage) string {
	footer := utils.FormatEventFooterContext(ctx, duration, model, usage)
	if footer == "" {
		return ""
	}
	switch ch {
	case Telegram:
		return "\n\n" + wrapBlock(ch, footer)
	default:
		return "\n" + wrapBlock(ch, footer)
	}
}

func AppendReplyFooter(ch Channel, text, footer string, hasMedia bool, execErrors []string) string {
	if hasMedia {
		footer = "🔗 " + footer
	}
	switch ch {
	case Telegram:
		text = fmt.Sprintf("%s\n\n%s", text, wrapBlock(ch, footer))
	default:
		text = fmt.Sprintf("%s\n%s", text, wrapBlock(ch, footer))
	}
	if len(execErrors) > 0 {
		errLine := wrapBlock(ch, "⚠️ "+strings.Join(execErrors, ", "))
		switch ch {
		case Telegram:
			text = fmt.Sprintf("%s\n\n%s", text, errLine)
		default:
			text = fmt.Sprintf("%s\n%s", text, errLine)
		}
	}
	return text
}
