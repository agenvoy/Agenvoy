package tui

import (
	"context"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	go_bot_telegram "github.com/pardnchiu/go-bot/core/telegram"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/telegram"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type TelegramAction struct {
	action string
}

type TelegramTokenSubmit struct {
	token string
}

type TelegramDone struct {
	action string
	err    error
}

func (t TUI) commandTelegram(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "enable", "disable":
			action := parts[1]
			return t, func() tea.Msg { return TelegramAction{action: action} }, true
		}
	}

	return t.commandChannel([]string{"channel", "telegram"})
}

func (t TUI) openTelegramTokenPrompt() (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    "Telegram Bot Token",
		input:    newPopupInput("", false),
		subtitle: "from @BotFather  Enter to submit  Esc to cancel",
		onConfirm: func(value string) any {
			return TelegramTokenSubmit{token: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func enableTelegram(token string) tea.Cmd {
	return func() tea.Msg {
		if token == "" {
			return TelegramDone{action: "enable", err: utilsmsg.New(utilsmsg.Invalid, "token", "is required")}
		}
		username, err := verifyTelegram(token)
		if err != nil {
			return TelegramDone{action: "enable", err: err}
		}
		if err := keychain.Set(telegram.Key, token); err != nil {
			return TelegramDone{action: "enable", err: utilsmsg.Wrap(utilsmsg.Internal, "keychain.Set", "", err)}
		}
		cfg, err := config.Load()
		if err != nil {
			return TelegramDone{action: "enable", err: utilsmsg.Trace("session.Load", "", err)}
		}
		cfg.TelegramEnabled = true
		cfg.TelegramUsername = username
		if err := config.Save(cfg); err != nil {
			return TelegramDone{action: "enable", err: utilsmsg.Trace("session.Save", "", err)}
		}
		return TelegramDone{action: "enable"}
	}
}

func disableTelegram() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return TelegramDone{action: "disable", err: utilsmsg.Trace("session.Load", "", err)}
		}
		if !cfg.TelegramEnabled && cfg.TelegramUsername == "" && keychain.Get(telegram.Key) == "" {
			return TelegramDone{action: "disable"}
		}
		if err := keychain.Delete(telegram.Key); err != nil {
			slog.Warn("keychain.Delete telegram token",
				slog.String("error", err.Error()))
		}
		cfg.TelegramEnabled = false
		cfg.TelegramUsername = ""
		if err := config.Save(cfg); err != nil {
			return TelegramDone{action: "disable", err: utilsmsg.Trace("session.Save", "", err)}
		}
		return TelegramDone{action: "disable"}
	}
}

func verifyTelegram(token string) (string, error) {
	client, err := go_bot_telegram.New(token)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "go_bot_telegram.New", "", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "go_bot_telegram.Start", "", err)
	}
	username := client.Status().Username
	_ = client.Close()
	return username, nil
}
