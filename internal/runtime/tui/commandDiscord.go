package tui

import (
	"context"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	go_bot_discord "github.com/pardnchiu/go-bot/core/discord"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/discord"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type DiscordAction struct {
	action string
}

type DiscordTokenSubmit struct {
	token string
}

type DiscordDone struct {
	action string
	err    error
}

func (t TUI) commandDiscord(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "enable", "disable":
			action := parts[1]
			return t, func() tea.Msg { return DiscordAction{action: action} }, true
		}
	}

	return t.commandChannel([]string{"channel", "discord"})
}

func (t TUI) openDiscordTokenPrompt() (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    "Discord Bot Token",
		input:    newPopupInput("", false),
		subtitle: "from Discord Developer Portal  Enter to submit  Esc to cancel",
		onConfirm: func(value string) any {
			return DiscordTokenSubmit{token: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func enableDiscord(token string) tea.Cmd {
	return func() tea.Msg {
		if token == "" {
			return DiscordDone{action: "enable", err: utilsmsg.New(utilsmsg.Invalid, "token", "is required")}
		}
		username, err := verifyDiscord(token)
		if err != nil {
			return DiscordDone{action: "enable", err: err}
		}
		if err := keychain.Set(discord.Key, token); err != nil {
			return DiscordDone{action: "enable", err: utilsmsg.Wrap(utilsmsg.Internal, "keychain.Set", "", err)}
		}
		cfg, err := config.Load()
		if err != nil {
			return DiscordDone{action: "enable", err: utilsmsg.Trace("session.Load", "", err)}
		}
		cfg.DiscordEnabled = true
		cfg.DiscordUsername = username
		if err := config.Save(cfg); err != nil {
			return DiscordDone{action: "enable", err: utilsmsg.Trace("session.Save", "", err)}
		}
		return DiscordDone{action: "enable"}
	}
}

func disableDiscord() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return DiscordDone{action: "disable", err: utilsmsg.Trace("session.Load", "", err)}
		}
		if !cfg.DiscordEnabled && cfg.DiscordUsername == "" && keychain.Get(discord.Key) == "" {
			return DiscordDone{action: "disable"}
		}
		if err := keychain.Delete(discord.Key); err != nil {
			slog.Warn("keychain.Delete discord token",
				slog.String("error", err.Error()))
		}
		cfg.DiscordEnabled = false
		cfg.DiscordUsername = ""
		if err := config.Save(cfg); err != nil {
			return DiscordDone{action: "disable", err: utilsmsg.Trace("session.Save", "", err)}
		}
		return DiscordDone{action: "disable"}
	}
}

func verifyDiscord(token string) (string, error) {
	client, err := go_bot_discord.New(token)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "go_bot_discord.New", "", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "go_bot_discord.Start", "", err)
	}
	username := client.Status().Username
	_ = client.Close()
	return username, nil
}
